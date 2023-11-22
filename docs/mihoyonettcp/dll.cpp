#include <Windows.h>

#include "common.h"
#include "dll.h"
#include "tcp.h"

typedef int* (*func_kcp_client_create)(KcpClientInitParams* param);
typedef void (*func_kcp_client_destroy)(int* kcp_client);
typedef int (*func_kcp_client_connect)(int* kcp_client, NativeKcpAddress* address, unsigned int token, unsigned int data);
typedef void (*func_kcp_client_disconnect)(int* kcp_client, unsigned int token, unsigned int data);
typedef int (*func_kcp_client_reconnect)(int* kcp_client, unsigned int token, unsigned int data);
typedef KcpPacket* (*func_kcp_packet_create)(unsigned char* data, int len);
typedef void (*func_kcp_packet_destroy)(KcpPacket* packet);
typedef int (*func_kcp_client_send_packet)(int* kcp_client, KcpPacket* packet);
typedef int (*func_kcp_client_try_deque_event)(int* kcp_client, KcpEvent* evt);
typedef void (*func_kcp_set_log_callback)(int* log_callback);
typedef void (*func_kcp_client_set_log_mask)(int* kcp_client, int log_mask);
typedef int (*func_kcp_client_network_thread)(int* kcp_client);
typedef int (*func_kcp_client_get_rtt)(int* kcp_client);
typedef int (*func_kcp_client_get_packet_loss)(int* kcp_client);
typedef int (*func_kcp_client_get_rx_bandwidth)(int* kcp_client);
typedef int (*func_kcp_client_get_tx_bandwidth)(int* kcp_client);

int* kcp_client_create(KcpClientInitParams* param)
{
	return tcp_client_create();
}

void kcp_client_destroy(int* kcp_client)
{
	return tcp_client_destroy(kcp_client);
}

int kcp_client_connect(int* kcp_client, NativeKcpAddress* address, unsigned int token, unsigned int data)
{
	if (address == NULL)
	{
		return -1;
	}
	return tcp_client_connect(kcp_client, address->host.host_v4, address->port);
}

void kcp_client_disconnect(int* kcp_client, unsigned int token, unsigned int data)
{
	return tcp_client_disconnect(kcp_client);
}

int kcp_client_reconnect(int* kcp_client, unsigned int token, unsigned int data)
{
	return 0;
}

KcpPacket* kcp_packet_create(unsigned char* data, int len)
{
	TcpPacket* tcp_packet = tcp_packet_create(data, len);
	if (tcp_packet == NULL)
	{
		return NULL;
	}
	KcpPacket* kcp_packet = new(KcpPacket);
	kcp_packet->data = tcp_packet->data;
	kcp_packet->dataLen = tcp_packet->dataLen;
	return kcp_packet;
}

void kcp_packet_destroy(KcpPacket* packet)
{
	if (packet == NULL)
	{
		return;
	}
	TcpPacket* tcp_packet = new(TcpPacket);
	tcp_packet->data = packet->data;
	tcp_packet->dataLen = packet->dataLen;
	tcp_packet_destroy(tcp_packet);
	return;
}

int kcp_client_send_packet(int* kcp_client, KcpPacket* packet)
{
	if (packet == NULL)
	{
		return -1;
	}
	TcpPacket* tcp_packet = new(TcpPacket);
	tcp_packet->data = packet->data;
	tcp_packet->dataLen = packet->dataLen;
	return tcp_client_send_packet(kcp_client, tcp_packet);
}

int kcp_client_try_deque_event(int* kcp_client, KcpEvent* evt)
{
	if (evt == NULL)
	{
		return -1;
	}
	TcpEvent* tcp_event = new(TcpEvent);
	int ret = tcp_client_try_deque_event(kcp_client, tcp_event);
	evt->type = (KcpEventType)tcp_event->type;
	evt->token = 1;
	evt->data = 0;
	evt->packet = NULL;
	switch (evt->type)
	{
	case TcpEventNotSet:
		break;
	case TcpEventConnect:
		evt->data = 1234567890;
		break;
	case TcpEventConnectFailed:
		evt->data = 1234567890;
		break;
	case TcpEventDisconnect:
		evt->data = 1;
		break;
	case TcpEventRecvMsg:
		if (tcp_event->packet != NULL)
		{
			KcpPacket* kcp_packet = new(KcpPacket);
			kcp_packet->data = tcp_event->packet->data;
			kcp_packet->dataLen = tcp_event->packet->dataLen;
			evt->packet = kcp_packet;
		}
		break;
	default:
		break;
	}
	return ret;
}

void kcp_set_log_callback(int* log_callback)
{
	return;
}

void kcp_client_set_log_mask(int* kcp_client, int log_mask)
{
	return;
}

int kcp_client_network_thread(int* kcp_client)
{
	return tcp_client_network_thread(kcp_client);
}

int kcp_client_get_rtt(int* kcp_client)
{
	return tcp_client_get_rtt(kcp_client);
}

int kcp_client_get_packet_loss(int* kcp_client)
{
	return 0;
}

int kcp_client_get_rx_bandwidth(int* kcp_client)
{
	return 0;
}

int kcp_client_get_tx_bandwidth(int* kcp_client)
{
	return 0;
}
