#pragma once

#include <stdio.h>
#include <time.h>

#define log(format, ...) log_print(__FILE__, __LINE__, __func__, format, ##__VA_ARGS__)

static inline void log_print(const char* file_name, int line, const char* func_name, const char* format, ...)
{
	time_t now = time(NULL);
	tm* tm_now = localtime(&now);
	char time_str[20];
	strftime(time_str, sizeof(time_str), "%Y-%m-%d %H:%M:%S", tm_now);

	printf("[%s] [mihoyonettcp] ", time_str);

	va_list args;
	va_start(args, format);
	vprintf(format, args);
	va_end(args);

	char short_file_name[64];
	memset(short_file_name, 0x00, 64);
	bool short_file_name_find = false;
	for (size_t i = strlen(file_name) - 1; i >= 0; i--)
	{
		if (file_name[i] == '\\')
		{
			if (strlen(file_name + i + 1) > 64)
			{
				break;
			}
			strcpy(short_file_name, file_name + i + 1);
			short_file_name_find = true;
			break;
		}
	}
	const char* print_file_name = short_file_name;
	if (!short_file_name_find)
	{
		print_file_name = file_name;
	}

	printf(" [%s:%d %s()]\n", print_file_name, line, func_name);
}
