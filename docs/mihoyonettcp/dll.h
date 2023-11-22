struct KcpClientInitParams
{
	unsigned long connection_timeout_time;
	unsigned int mtu;
	unsigned int snd_wnd;
	unsigned int rcv_wnd;
	int kcp_log_mask;
};

struct AllHostAddress
{
	unsigned char is_ipv6;
	unsigned int host_v4;
	unsigned char host_v6[16];
};

struct NativeKcpAddress
{
	AllHostAddress host;
	unsigned short port;
};

struct KcpPacket
{
	int* data;
	unsigned int dataLen;
};

enum KcpEventType
{
	EventNotSet = -1,
	EventConnect = 0,
	EventConnectFailed = 1,
	EventDisconnect = 2,
	EventRecvMsg = 3,
	EventCount = 4
};

struct KcpEvent
{
	KcpEventType type;
	unsigned int token;
	unsigned int data;
	KcpPacket* packet;
};

extern "C"
{
	__declspec(dllexport) int* kcp_client_create(KcpClientInitParams* param);
	__declspec(dllexport) void kcp_client_destroy(int* kcp_client);
	__declspec(dllexport) int kcp_client_connect(int* kcp_client, NativeKcpAddress* address, unsigned int token, unsigned int data);
	__declspec(dllexport) void kcp_client_disconnect(int* kcp_client, unsigned int token, unsigned int data);
	__declspec(dllexport) int kcp_client_reconnect(int* kcp_client, unsigned int token, unsigned int data);
	__declspec(dllexport) KcpPacket* kcp_packet_create(unsigned char* data, int len);
	__declspec(dllexport) void kcp_packet_destroy(KcpPacket* packet);
	__declspec(dllexport) int kcp_client_send_packet(int* kcp_client, KcpPacket* packet);
	__declspec(dllexport) int kcp_client_try_deque_event(int* kcp_client, KcpEvent* evt);
	__declspec(dllexport) void kcp_set_log_callback(int* log_callback);
	__declspec(dllexport) void kcp_client_set_log_mask(int* kcp_client, int log_mask);
	__declspec(dllexport) int kcp_client_network_thread(int* kcp_client);
	__declspec(dllexport) int kcp_client_get_rtt(int* kcp_client);
	__declspec(dllexport) int kcp_client_get_packet_loss(int* kcp_client);
	__declspec(dllexport) int kcp_client_get_rx_bandwidth(int* kcp_client);
	__declspec(dllexport) int kcp_client_get_tx_bandwidth(int* kcp_client);
}
