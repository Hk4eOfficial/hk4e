#pragma once

struct TcpPacket
{
	int* data;
	unsigned int dataLen;
};

enum TcpEventType
{
	TcpEventNotSet = -1,
	TcpEventConnect = 0,
	TcpEventConnectFailed = 1,
	TcpEventDisconnect = 2,
	TcpEventRecvMsg = 3,
	TcpEventCount = 4
};

struct TcpEvent
{
	TcpEventType type;
	TcpPacket* packet;
};

int* tcp_client_create();
void tcp_client_destroy(int* tcp_client);
int tcp_client_connect(int* tcp_client, unsigned int ip, unsigned short port);
void tcp_client_disconnect(int* tcp_client);
TcpPacket* tcp_packet_create(unsigned char* data, int len);
void tcp_packet_destroy(TcpPacket* packet);
int tcp_client_send_packet(int* tcp_client, TcpPacket* packet);
int tcp_client_try_deque_event(int* tcp_client, TcpEvent* evt);
int tcp_client_network_thread(int* tcp_client);
int tcp_client_get_rtt(int* tcp_client);
