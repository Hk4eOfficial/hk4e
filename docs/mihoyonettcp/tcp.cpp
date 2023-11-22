#include <WinSock2.h>
#include <sys/timeb.h>

#include "common.h"
#include "tcp.h"

#pragma comment(lib, "ws2_32.lib")

#define MAX_PAYLOAD_LEN 343*1024
#define TCP_NO_DELAY true

struct TcpEventQueue
{
	TcpEvent** RingArray;
	unsigned int RingArrayLen;
	unsigned int HeadPtr;
	unsigned int TailPtr;
	unsigned int Len;
};

TcpEventQueue* tcp_event_queue_create(int size)
{
	TcpEventQueue* r = new(TcpEventQueue);
	r->RingArray = (TcpEvent**)malloc(sizeof(TcpEvent*) * size);
	r->RingArrayLen = size;
	r->HeadPtr = 0;
	r->TailPtr = 0;
	r->Len = 0;
	return r;
}

void tcp_event_queue_destroy(TcpEventQueue* queue)
{
	if (queue != NULL && queue->RingArray != NULL)
	{
		free(queue->RingArray);
	}
}

int tcp_event_queue_len(TcpEventQueue* queue)
{
	return queue->Len;
}

void tcp_event_queue_enque(TcpEventQueue* queue, TcpEvent* tcp_event)
{
	if (queue->Len >= queue->RingArrayLen)
	{
		return;
	}
	queue->RingArray[queue->TailPtr] = tcp_event;
	queue->TailPtr++;
	if (queue->TailPtr >= queue->RingArrayLen)
	{
		queue->TailPtr = 0;
	}
	queue->Len++;
}

TcpEvent* tcp_event_queue_deque(TcpEventQueue* queue)
{
	if (queue->Len == 0)
	{
		return NULL;
	}
	TcpEvent* ret = queue->RingArray[queue->HeadPtr];
	queue->HeadPtr++;
	if (queue->HeadPtr >= queue->RingArrayLen)
	{
		queue->HeadPtr = 0;
	}
	queue->Len--;
	return ret;
}

TcpEventQueue* tcp_event_queue = NULL;

unsigned char* global_tcp_recv_buffer = NULL;
unsigned int global_tcp_recv_buffer_len = 0;

unsigned char* tcp_recv_buffer = NULL;

unsigned int rtt = 0;
time_t rtt_last_send_time = 0;

int* tcp_client_create()
{
	WSADATA wsadata;
	if (WSAStartup(MAKEWORD(2, 2), &wsadata) != 0)
	{
		log("WSAStartup error");
		return NULL;
	}

	SOCKET tcp_client = socket(AF_INET, SOCK_STREAM, IPPROTO_TCP);
	if (tcp_client == INVALID_SOCKET)
	{
		log("socket error");
		WSACleanup();
		return NULL;
	}

	tcp_event_queue = tcp_event_queue_create(1024);
	global_tcp_recv_buffer = (unsigned char*)malloc(MAX_PAYLOAD_LEN);
	tcp_recv_buffer = (unsigned char*)malloc(MAX_PAYLOAD_LEN);

	log("tcp_client_create ok");

	return (int*)tcp_client;
}

void tcp_client_destroy(int* tcp_client)
{
	tcp_event_queue_destroy(tcp_event_queue);
	free(global_tcp_recv_buffer);
	global_tcp_recv_buffer_len = 0;
	free(tcp_recv_buffer);
	rtt = 0;
	rtt_last_send_time = 0;
	closesocket((SOCKET)tcp_client);
	WSACleanup();

	log("tcp_client_destroy ok");

	return;
}

int tcp_client_connect(int* tcp_client, unsigned int ip, unsigned short port)
{
	SOCKADDR_IN addr;
	addr.sin_family = AF_INET;
	addr.sin_addr.S_un.S_addr = ip;
	addr.sin_port = htons(port);

	if (connect((SOCKET)tcp_client, (SOCKADDR*)&addr, sizeof(SOCKADDR)) == SOCKET_ERROR)
	{
		log("connect error");
		TcpEvent* tcp_event = new(TcpEvent);
		tcp_event->type = TcpEventConnectFailed;
		tcp_event->packet = NULL;
		tcp_event_queue_enque(tcp_event_queue, tcp_event);
		return -1;
	}

	if (TCP_NO_DELAY)
	{
		int flag = 1;
		if (setsockopt((SOCKET)tcp_client, IPPROTO_TCP, TCP_NODELAY, (const char*)(&flag), sizeof(int)) != 0)
		{
			log("setsockopt error");
			return -2;
		}
	}

	unsigned long mode = 1;
	if (ioctlsocket((SOCKET)tcp_client, FIONBIO, &mode) != 0)
	{
		log("ioctlsocket error");
		return -3;
	}

	TcpEvent* tcp_event = new(TcpEvent);
	tcp_event->type = TcpEventConnect;
	tcp_event->packet = NULL;
	tcp_event_queue_enque(tcp_event_queue, tcp_event);

	log("tcp_client_connect ok, ip: %u, port: %hu", ip, port);

	return 0;
}

void tcp_client_disconnect(int* tcp_client)
{
	shutdown((SOCKET)tcp_client, SD_BOTH);

	log("tcp_client_disconnect ok");

	return;
}

TcpPacket* tcp_packet_create(unsigned char* data, int len)
{
	TcpPacket* tcp_packet = new(TcpPacket);
	tcp_packet->data = (int*)malloc(len);
	if (tcp_packet->data == NULL)
	{
		return NULL;
	}
	memcpy(tcp_packet->data, data, len);
	tcp_packet->dataLen = len;
	return tcp_packet;
}

void tcp_packet_destroy(TcpPacket* tcp_packet)
{
	if (tcp_packet != NULL && tcp_packet->data != NULL)
	{
		free(tcp_packet->data);
	}
	return;
}

int tcp_client_send_packet(int* tcp_client, TcpPacket* tcp_packet)
{
	if (tcp_packet == NULL)
	{
		return -1;
	}
	unsigned long data_len = htonl(tcp_packet->dataLen);
	int r1 = send((SOCKET)tcp_client, (const char*)&data_len, 4, 0);
	int r2 = send((SOCKET)tcp_client, (const char*)tcp_packet->data, tcp_packet->dataLen, 0);
	tcp_packet_destroy(tcp_packet);
	if (r1 <= 0 || r2 <= 0)
	{
		TcpEvent* tcp_event = new(TcpEvent);
		tcp_event->type = TcpEventDisconnect;
		tcp_event->packet = NULL;
		tcp_event_queue_enque(tcp_event_queue, tcp_event);
		return -2;
	}
	timeb t;
	ftime(&t);
	time_t now = t.time * 1000 + t.millitm;
	if (now - rtt_last_send_time > 1000)
	{
		unsigned int rtt_data = 0;
		int r = send((SOCKET)tcp_client, (const char*)(&rtt_data), 4, 0);
		if (r <= 0)
		{
			TcpEvent* tcp_event = new(TcpEvent);
			tcp_event->type = TcpEventDisconnect;
			tcp_event->packet = NULL;
			tcp_event_queue_enque(tcp_event_queue, tcp_event);
			return 0;
		}
		rtt_last_send_time = now;
	}
	return 0;
}

int tcp_client_try_deque_event(int* tcp_client, TcpEvent* evt)
{
	evt->type = TcpEventNotSet;
	evt->packet = NULL;
	if (tcp_event_queue_len(tcp_event_queue) == 0)
	{
		return -1;
	}
	if (evt == NULL)
	{
		return -2;
	}
	TcpEvent* tcp_event = tcp_event_queue_deque(tcp_event_queue);
	evt->type = tcp_event->type;
	evt->packet = tcp_event->packet;
	return 0;
}

int tcp_client_network_thread(int* tcp_client)
{
	int recv_len = recv((SOCKET)tcp_client, (char*)tcp_recv_buffer, MAX_PAYLOAD_LEN, 0);
	if (global_tcp_recv_buffer_len > 0 || recv_len > 0)
	{
		int loop_times = 0;
		while (true)
		{
			loop_times++;
			if (loop_times > 1000)
			{
				log("tcp recv loop bug");
				TcpEvent* tcp_event = new(TcpEvent);
				tcp_event->type = TcpEventDisconnect;
				tcp_event->packet = NULL;
				tcp_event_queue_enque(tcp_event_queue, tcp_event);
				return 0;
			}
			if (recv_len > 0)
			{
				memcpy(global_tcp_recv_buffer + global_tcp_recv_buffer_len, tcp_recv_buffer, recv_len);
				global_tcp_recv_buffer_len += recv_len;
				recv_len = 0;
			}
			if (global_tcp_recv_buffer_len < 4)
			{
				return 0;
			}
			unsigned int payload_len = ntohl(*(unsigned int*)global_tcp_recv_buffer);
			if (payload_len == 0)
			{
				global_tcp_recv_buffer_len -= 4;
				memcpy(global_tcp_recv_buffer, global_tcp_recv_buffer + 4, global_tcp_recv_buffer_len);
				timeb t;
				ftime(&t);
				time_t now = t.time * 1000 + t.millitm;
				rtt = (unsigned int)(now - rtt_last_send_time);
				return 0;
			}
			if (payload_len == 0xffffffff)
			{
				global_tcp_recv_buffer_len -= 4;
				memcpy(global_tcp_recv_buffer, global_tcp_recv_buffer + 4, global_tcp_recv_buffer_len);
				unsigned int rtt_data = 0xffffffff;
				int r = send((SOCKET)tcp_client, (const char*)(&rtt_data), 4, 0);
				if (r <= 0)
				{
					TcpEvent* tcp_event = new(TcpEvent);
					tcp_event->type = TcpEventDisconnect;
					tcp_event->packet = NULL;
					tcp_event_queue_enque(tcp_event_queue, tcp_event);
					return 0;
				}
				return 0;
			}
			if (global_tcp_recv_buffer_len - 4 < payload_len)
			{
				return 0;
			}
			TcpPacket* tcp_packet = tcp_packet_create(global_tcp_recv_buffer + 4, payload_len);
			TcpEvent* tcp_event = new(TcpEvent);
			tcp_event->type = TcpEventRecvMsg;
			tcp_event->packet = tcp_packet;
			tcp_event_queue_enque(tcp_event_queue, tcp_event);
			global_tcp_recv_buffer_len -= payload_len + 4;
			memcpy(global_tcp_recv_buffer, global_tcp_recv_buffer + payload_len + 4, global_tcp_recv_buffer_len);
		}
	}
	else if (recv_len == 0)
	{
		TcpEvent* tcp_event = new(TcpEvent);
		tcp_event->type = TcpEventDisconnect;
		tcp_event->packet = NULL;
		tcp_event_queue_enque(tcp_event_queue, tcp_event);
	}
	else if (recv_len < 0)
	{
		return 0;
	}
	return 0;
}

int tcp_client_get_rtt(int* tcp_client)
{
	return rtt;
}
