#ifndef _BIND_H_
#define _BIND_H_

struct bind_event_t {
    struct kevent_t event;
    struct process_context_t process;
    struct span_context_t span;
    struct container_context_t container;
    struct syscall_t syscall;

    int socket;
    /* TODO */
};

/* #include <linux/types.h> */
/* #include <sys/socket.h> */
/* SYSCALL_KPROBE3(bind, int, socket, struct sockaddr_in*, addr, socklen_t, addr_len) { */
SYSCALL_KPROBE3(bind, int, socket, void*, addr, int, addr_len) {
    struct policy_t policy = fetch_policy(EVENT_BIND);
    if (is_discarded_by_process(policy.mode, EVENT_BIND)) {
        return 0;
    }

    /* cache the bind and wait to grab the retval to send it */
    struct syscall_cache_t syscall = {
        .type = EVENT_BIND,
        .bind = {
            .socket = socket,
            .addr = addr,
            .addr_len = addr_len,
        },
    };
    cache_syscall(&syscall);
}

SYSCALL_KRETPROBE(bind) {
    struct syscall_cache_t *syscall = pop_syscall(EVENT_BIND);
    if (!syscall) {
        return 0;
    }

    int retval = PT_REGS_RC(ctx);
    if (IS_UNHANDLED_ERROR(retval)) {
        return 0;
    }

    struct bind_event_t event = {
        .syscall.retval = retval,
        .socket = syscall->bind.socket,
        /* TODO */
    };

    struct proc_cache_t *entry = fill_process_context(&event.process);
    fill_container_context(entry, &event.container);
    fill_span_context(&event.span);
    send_event(ctx, EVENT_BIND, event);
    return 0;
}


#endif /* _BIND_H_ */
