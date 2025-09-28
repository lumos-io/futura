//go:build ignore

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_core_read.h>

// Security metrics per container
struct security_metrics {
    // Syscall monitoring
    __u64 total_syscalls;
    __u64 suspicious_syscalls;
    __u64 privilege_escalation_attempts;
    __u64 syscall_rate_violations;

    // Resource violations
    __u64 memory_violations;
    __u64 cpu_violations;
    __u64 fd_violations;
    __u64 process_violations;

    // Network security
    __u64 suspicious_connections;
    __u64 blocked_connections;
    __u64 port_scan_attempts;

    // Process security
    __u64 new_processes;
    __u64 suspicious_processes;
    __u64 setuid_executions;
    __u64 container_escape_attempts;

    // File system security
    __u64 unauthorized_file_access;
    __u64 sensitive_file_access;
    __u64 system_file_modifications;

    __u64 last_update;
};

// Syscall frequency tracking
struct syscall_stats {
    __u64 count;
    __u64 last_seen;
    __u64 avg_frequency;
    __u8 is_suspicious;
};

// Process tracking for security
struct process_info {
    __u32 pid;
    __u32 ppid;
    __u32 uid;
    __u32 gid;
    __u64 start_time;
    char comm[16];
    char filename[256];
    __u8 is_suspicious;
    __u64 cgroup_id;
};

// Network connection tracking
struct network_connection {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8 protocol;
    __u64 timestamp;
    __u64 bytes_transferred;
    __u8 is_suspicious;
    __u64 cgroup_id;
};

// Security event for real-time alerting
struct security_event {
    __u64 timestamp;
    __u64 cgroup_id;
    __u32 pid;
    __u32 event_type; // 0=syscall_anomaly, 1=resource_violation, 2=network_anomaly, 3=process_anomaly, 4=file_anomaly
    __u32 severity;   // 0=low, 1=medium, 2=high, 3=critical
    __u64 value1;
    __u64 value2;
    char description[256];
};

// Maps
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, __u64);     // cgroup_id
    __type(value, struct security_metrics);
} security_metrics_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1000);
    __type(key, __u32);     // syscall_number
    __type(value, struct syscall_stats);
} syscall_stats_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, __u32);     // pid
    __type(value, struct process_info);
} process_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 5000);
    __type(key, __u64);     // connection_hash
    __type(value, struct network_connection);
} network_connections_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} security_events SEC(".maps");

// Allowlist for normal syscalls (simplified)
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 500);
    __type(key, __u32);     // syscall_number
    __type(value, __u8);    // allowed (1) or not (0)
} syscall_allowlist SEC(".maps");

// Sensitive file paths
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1000);
    __type(key, __u64);     // path_hash
    __type(value, __u8);    // sensitivity_level
} sensitive_paths_map SEC(".maps");

// Helper to get cgroup ID
static __always_inline __u64 get_cgroup_id(void) {
    struct task_struct *task = (struct task_struct *)bpf_get_current_task();
    return BPF_CORE_READ(task, cgroups, dfl_cgrp, kn, id);
}

// Helper to check if syscall is suspicious
static __always_inline int is_suspicious_syscall(__u32 syscall_nr) {
    // Check against allowlist
    __u8 *allowed = bpf_map_lookup_elem(&syscall_allowlist, &syscall_nr);
    if (allowed && *allowed == 1) {
        return 0; // Normal syscall
    }

    // Check for privilege escalation syscalls
    switch (syscall_nr) {
        case 105: // setuid
        case 106: // setgid
        case 113: // setreuid
        case 114: // setregid
        case 117: // setresuid
        case 119: // setresgid
        case 157: // prctl
        case 158: // arch_prctl
        case 163: // acct
        case 164: // settimeofday
        case 165: // mount
        case 166: // umount2
        case 167: // swapon
        case 168: // swapoff
        case 169: // reboot
        case 172: // iopl
        case 173: // ioperm
        case 310: // process_vm_readv
        case 311: // process_vm_writev
            return 1; // Suspicious
        default:
            return 0;
    }
}

// Helper to hash file path
static __always_inline __u64 hash_path(const char *path) {
    __u64 hash = 5381;
    for (int i = 0; i < 256 && path[i] != '\0'; i++) {
        hash = ((hash << 5) + hash) + path[i];
    }
    return hash;
}

// Helper to check if string contains substring (simplified)
static __always_inline int str_contains(const char *str, const char *substr) {
    if (!str || !substr) return 0;

    for (int i = 0; i < 256 && str[i] != '\0'; i++) {
        int match = 1;
        for (int j = 0; j < 32 && substr[j] != '\0'; j++) {
            if (i + j >= 256 || str[i + j] != substr[j]) {
                match = 0;
                break;
            }
        }
        if (match) return 1;
    }
    return 0;
}

// Helper to emit security event
static __always_inline void emit_security_event(__u64 cgroup_id, __u32 pid, __u32 event_type,
                                               __u32 severity, __u64 value1, __u64 value2,
                                               const char *description) {
    struct security_event *event = bpf_ringbuf_reserve(&security_events, sizeof(*event), 0);
    if (event) {
        event->timestamp = bpf_ktime_get_ns();
        event->cgroup_id = cgroup_id;
        event->pid = pid;
        event->event_type = event_type;
        event->severity = severity;
        event->value1 = value1;
        event->value2 = value2;
        bpf_probe_read_kernel_str(event->description, sizeof(event->description), description);
        bpf_ringbuf_submit(event, 0);
    }
}

// Raw tracepoint for all syscalls
SEC("raw_tracepoint/sys_enter")
int trace_sys_enter(struct bpf_raw_tracepoint_args *ctx) {
    __u32 syscall_nr = (__u32)ctx->args[1];
    __u64 cgroup_id = get_cgroup_id();
    __u32 pid = bpf_get_current_pid_tgid() >> 32;
    __u64 now = bpf_ktime_get_ns();

    // Update security metrics
    struct security_metrics *metrics = bpf_map_lookup_elem(&security_metrics_map, &cgroup_id);
    if (!metrics) {
        struct security_metrics new_metrics = {0};
        new_metrics.total_syscalls = 1;
        new_metrics.last_update = now;
        bpf_map_update_elem(&security_metrics_map, &cgroup_id, &new_metrics, BPF_ANY);
        metrics = bpf_map_lookup_elem(&security_metrics_map, &cgroup_id);
    } else {
        __sync_fetch_and_add(&metrics->total_syscalls, 1);
        metrics->last_update = now;
    }

    // Check for suspicious syscalls
    if (is_suspicious_syscall(syscall_nr)) {
        __sync_fetch_and_add(&metrics->suspicious_syscalls, 1);

        // Check if this is a privilege escalation attempt
        switch (syscall_nr) {
            case 105: // setuid
            case 106: // setgid
            case 113: // setreuid
            case 114: // setregid
            case 117: // setresuid
            case 119: // setresgid:
                __sync_fetch_and_add(&metrics->privilege_escalation_attempts, 1);
                emit_security_event(cgroup_id, pid, 0, 2, syscall_nr, 0, "Privilege escalation attempt detected");
                break;
        }

        emit_security_event(cgroup_id, pid, 0, 1, syscall_nr, 0, "Suspicious syscall detected");
    }

    // Update syscall frequency stats
    struct syscall_stats *stats = bpf_map_lookup_elem(&syscall_stats_map, &syscall_nr);
    if (!stats) {
        struct syscall_stats new_stats = {
            .count = 1,
            .last_seen = now,
            .avg_frequency = 0,
            .is_suspicious = is_suspicious_syscall(syscall_nr)
        };
        bpf_map_update_elem(&syscall_stats_map, &syscall_nr, &new_stats, BPF_ANY);
    } else {
        __sync_fetch_and_add(&stats->count, 1);
        stats->last_seen = now;
    }

    return 0;
}

// Tracepoint for process creation
SEC("tracepoint/sched/sched_process_exec")
int trace_process_exec(struct trace_event_raw_sched_process_exec *ctx) {
    __u32 pid = bpf_get_current_pid_tgid() >> 32;
    __u64 cgroup_id = get_cgroup_id();
    __u64 now = bpf_ktime_get_ns();

    struct task_struct *task = (struct task_struct *)bpf_get_current_task();

    struct process_info proc = {0};
    proc.pid = pid;
    proc.ppid = BPF_CORE_READ(task, real_parent, pid);
    proc.uid = BPF_CORE_READ(task, cred, uid.val);
    proc.gid = BPF_CORE_READ(task, cred, gid.val);
    proc.start_time = now;
    proc.cgroup_id = cgroup_id;

    bpf_get_current_comm(&proc.comm, sizeof(proc.comm));
    // Get the executable path from the task struct
    struct file *file = BPF_CORE_READ(task, mm, exe_file);
    if (file) {
        struct dentry *dentry = BPF_CORE_READ(file, f_path.dentry);
        if (dentry) {
            bpf_probe_read_kernel_str(proc.filename, sizeof(proc.filename), BPF_CORE_READ(dentry, d_name.name));
        }
    }

    // Check for suspicious process characteristics
    proc.is_suspicious = 0;

    // Check for suspicious binaries (simplified detection)
    char *filename = proc.filename;
    if (filename) {
        // Look for common attack tools or suspicious names
        if (str_contains(filename, "ncat") || str_contains(filename, "nc") ||
            str_contains(filename, "nmap") || str_contains(filename, "wget") ||
            str_contains(filename, "curl") || str_contains(filename, "python") ||
            str_contains(filename, "perl") || str_contains(filename, "ruby")) {
            proc.is_suspicious = 1;
        }
    }

    // Check for setuid/setgid executables
    if (proc.uid == 0 && proc.pid != 1) { // Running as root but not init
        proc.is_suspicious = 1;
    }

    bpf_map_update_elem(&process_map, &pid, &proc, BPF_ANY);

    // Update security metrics
    struct security_metrics *metrics = bpf_map_lookup_elem(&security_metrics_map, &cgroup_id);
    if (metrics) {
        __sync_fetch_and_add(&metrics->new_processes, 1);
        if (proc.is_suspicious) {
            __sync_fetch_and_add(&metrics->suspicious_processes, 1);
            emit_security_event(cgroup_id, pid, 3, 1, proc.uid, 0, "Suspicious process detected");
        }
        if (proc.uid == 0) {
            __sync_fetch_and_add(&metrics->setuid_executions, 1);
            emit_security_event(cgroup_id, pid, 3, 2, proc.uid, 0, "Root process execution");
        }
    }

    return 0;
}

// Tracepoint for file operations
SEC("tracepoint/syscalls/sys_enter_openat")
int trace_file_access(struct trace_event_raw_sys_enter *ctx) {
    __u64 cgroup_id = get_cgroup_id();
    __u32 pid = bpf_get_current_pid_tgid() >> 32;

    // Get filename from syscall arguments
    char filename[256];
    bpf_probe_read_user_str(filename, sizeof(filename), (void *)ctx->args[1]);

    __u64 path_hash = hash_path(filename);

    // Check if this is a sensitive file
    __u8 *sensitivity = bpf_map_lookup_elem(&sensitive_paths_map, &path_hash);
    if (sensitivity && *sensitivity > 0) {
        struct security_metrics *metrics = bpf_map_lookup_elem(&security_metrics_map, &cgroup_id);
        if (metrics) {
            __sync_fetch_and_add(&metrics->sensitive_file_access, 1);
            emit_security_event(cgroup_id, pid, 4, *sensitivity, 0, 0, "Sensitive file access detected");
        }
    }

    // Check for access to system files
    if (str_contains(filename, "/etc/") || str_contains(filename, "/sys/") ||
        str_contains(filename, "/proc/") || str_contains(filename, "/dev/")) {
        struct security_metrics *metrics = bpf_map_lookup_elem(&security_metrics_map, &cgroup_id);
        if (metrics) {
            __sync_fetch_and_add(&metrics->system_file_modifications, 1);
            emit_security_event(cgroup_id, pid, 4, 1, 0, 0, "System file access detected");
        }
    }

    return 0;
}

// Kprobe for network security monitoring
SEC("kprobe/tcp_connect")
int trace_tcp_connect(struct pt_regs *ctx) {
    __u64 cgroup_id = get_cgroup_id();
    __u32 pid = bpf_get_current_pid_tgid() >> 32;

    // This would analyze the connection details in a real implementation
    // For now, we'll just count connections
    struct security_metrics *metrics = bpf_map_lookup_elem(&security_metrics_map, &cgroup_id);
    if (metrics) {
        // Simple heuristic: too many connections might be suspicious
        if (metrics->suspicious_connections > 100) {
            emit_security_event(cgroup_id, pid, 2, 1, 0, 0, "High connection rate detected");
        }
    }

    return 0;
}

// Tracepoint for memory pressure events
SEC("tracepoint/vmscan/mm_vmscan_memcg_softlimit_reclaim_begin")
int trace_memory_pressure(void *ctx) {
    __u64 cgroup_id = get_cgroup_id();
    __u32 pid = bpf_get_current_pid_tgid() >> 32;

    struct security_metrics *metrics = bpf_map_lookup_elem(&security_metrics_map, &cgroup_id);
    if (metrics) {
        __sync_fetch_and_add(&metrics->memory_violations, 1);
        emit_security_event(cgroup_id, pid, 1, 1, 0, 0, "Memory pressure event detected");
    }

    return 0;
}

// Tracepoint for OOM kills
SEC("tracepoint/oom/oom_score_adj_update")
int trace_oom_event(void *ctx) {
    __u64 cgroup_id = get_cgroup_id();
    __u32 pid = bpf_get_current_pid_tgid() >> 32;

    struct security_metrics *metrics = bpf_map_lookup_elem(&security_metrics_map, &cgroup_id);
    if (metrics) {
        __sync_fetch_and_add(&metrics->memory_violations, 1);
        emit_security_event(cgroup_id, pid, 1, 2, 0, 0, "OOM event detected");
    }

    return 0;
}

char _license[] SEC("license") = "GPL";