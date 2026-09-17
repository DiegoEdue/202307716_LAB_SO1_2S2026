//go:build ignore

// Sonda eBPF
// Se dispara cada vez que CUALQUIER proceso invoca kill(2), incluyendo cuando el Daemon termina un contenedor.

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>

char __license[] SEC("license") = "Dual MIT/GPL";

struct kill_event {
    __u32 sender_pid;
    __u32 target_pid;
    __s32 signal;
};

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} events SEC(".maps");

SEC("tracepoint/syscalls/sys_enter_kill")
int handle_kill(struct trace_event_raw_sys_enter *ctx)
{
    struct kill_event *evt;

    evt = bpf_ringbuf_reserve(&events, sizeof(*evt), 0);
    if (!evt)
        return 0;

    evt->sender_pid = bpf_get_current_pid_tgid() >> 32;
    evt->target_pid = (__u32)ctx->args[0];
    evt->signal     = (__s32)ctx->args[1];

    bpf_ringbuf_submit(evt, 0);
    return 0;
}