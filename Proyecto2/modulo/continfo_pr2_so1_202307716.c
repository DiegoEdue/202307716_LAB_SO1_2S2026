#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/string.h>
#include <linux/init.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/mm.h>
#include <linux/sched.h>
#include <linux/timer.h>
#include <linux/jiffies.h>
#include <linux/uaccess.h>
#include <linux/tty.h>
#include <linux/sched/signal.h>
#include <linux/fs.h>
#include <linux/slab.h>
#include <linux/sched/mm.h>
#include <linux/binfmts.h>
#include <linux/timekeeping.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Diego");
MODULE_DESCRIPTION("Proyecto 2 SO1");
MODULE_VERSION("1.0");

#define PROC_NAME    "continfo_pr2_so1_202307716"

#define MAX_CMDLINE_LENGTH 256
#define SHIM_COMM          "containerd-shim"
#define CONTAINER_ID_LEN   72

static char *get_process_cmdline(struct task_struct *task)
{
    struct mm_struct *mm;
    char *cmdline, *p;
    unsigned long arg_start, arg_end;
    long len;
    int i;

    cmdline = kmalloc(MAX_CMDLINE_LENGTH, GFP_KERNEL);
    if (!cmdline)
        return NULL;

    mm = get_task_mm(task);
    if (!mm) {
        kfree(cmdline);
        return NULL;
    }

    down_read(&mm->mmap_lock);
    arg_start = mm->arg_start;
    arg_end = mm->arg_end;
    up_read(&mm->mmap_lock);

    len = (long)(arg_end - arg_start);
    if (len <= 0) {
        mmput(mm);
        kfree(cmdline);
        return NULL;
    }
    if (len > MAX_CMDLINE_LENGTH - 1)
        len = MAX_CMDLINE_LENGTH - 1;

    if (access_process_vm(task, arg_start, cmdline, len, 0) != len) {
        mmput(mm);
        kfree(cmdline);
        return NULL;
    }

    cmdline[len] = '\0';
    p = cmdline;
    for (i = 0; i < len; i++)
        if (p[i] == '\0')
            p[i] = ' ';

    mmput(mm);
    return cmdline;
}

static int extract_container_id(const char *cmdline, char *out, size_t out_len)
{
    const char *pos = strstr(cmdline, "-id ");
    const char *start, *end;
    size_t len;

    if (!pos)
        return 0;

    start = pos + 4;
    end = start;
    while (*end && *end != ' ')
        end++;

    len = end - start;
    if (len == 0 || len >= out_len)
        return 0;

    memcpy(out, start, len);
    out[len] = '\0';
    return 1;
}

/* Escapa un string arbitrario, antes de meterlo en el JSON. Sin esto, 
   cualquier proceso del sistema con uno de esos
 * caracteres rompe el archivo /proc completo. */
static void seq_print_json_escaped(struct seq_file *m, const char *s)
{
    if (!s) {
        seq_printf(m, "N/A");
        return;
    }
    for (; *s; s++) {
        unsigned char c = *s;
        switch (c) {
        case '"':
            seq_printf(m, "\\\"");
            break;
        case '\\':
            seq_printf(m, "\\\\");
            break;
        case '\n':
            seq_printf(m, "\\n");
            break;
        case '\r':
            seq_printf(m, "\\r");
            break;
        case '\t':
            seq_printf(m, "\\t");
            break;
        default:
            if (c < 0x20)
                seq_printf(m, "\\u%04x", c);
            else
                seq_putc(m, c);
        }
    }
}

static int sysinfo_show(struct seq_file *m, void *v)
{
    struct sysinfo si;
    struct task_struct *task;
    unsigned long total_jiffies = jiffies;
    unsigned long totalram_kb, freeram_kb, usedram_kb;
    int first_process = 1;

    si_meminfo(&si);
    totalram_kb = si.totalram << (PAGE_SHIFT - 10);
    freeram_kb  = si.freeram  << (PAGE_SHIFT - 10);
    usedram_kb  = totalram_kb - freeram_kb;

    seq_printf(m, "{\n");
    seq_printf(m, "  \"TotalRAM_KB\": %lu,\n", totalram_kb);
    seq_printf(m, "  \"FreeRAM_KB\": %lu,\n", freeram_kb);
    seq_printf(m, "  \"UsedRAM_KB\": %lu,\n", usedram_kb);
    seq_printf(m, "  \"Processes\": [\n");

    rcu_read_lock();
    for_each_process(task) {
        struct mm_struct *mm = task->mm;
        unsigned long vsz = 0, rss = 0, mem_usage = 0, cpu_usage = 0;
        unsigned long total_time;
        char *own_cmdline = NULL;
        char container_id[CONTAINER_ID_LEN];
        int is_container = 0;

        if (!mm)
            continue;

        vsz = mm->total_vm << (PAGE_SHIFT - 10);
        rss = get_mm_rss(mm) << (PAGE_SHIFT - 10);
        if (totalram_kb > 0)
            mem_usage = (rss * 1000) / totalram_kb;

        total_time = task->utime + task->stime;
        if (total_jiffies > 0)
            cpu_usage = (total_time * 10000) / total_jiffies;

        {
            struct task_struct *parent = task->real_parent;

            if (parent && strncmp(parent->comm, SHIM_COMM, strlen(SHIM_COMM)) == 0) {
                char *parent_cmdline = get_process_cmdline(parent);

                if (parent_cmdline) {
                    if (extract_container_id(parent_cmdline, container_id, sizeof(container_id)))
                        is_container = 1;
                    kfree(parent_cmdline);
                }
            }
        }

        if (!is_container)
            own_cmdline = get_process_cmdline(task);

        if (!first_process)
            seq_printf(m, ",\n");
        else
            first_process = 0;

        seq_printf(m, "     {\n");
        seq_printf(m, "        \"PID\": %d,\n", task->pid);

        seq_printf(m, "        \"Name\": \"");
        seq_print_json_escaped(m, task->comm);
        seq_printf(m, "\",\n");

        if (is_container) {
            seq_printf(m, "        \"ContainerID\": \"");
            seq_print_json_escaped(m, container_id);
            seq_printf(m, "\",\n");
        } else {
            seq_printf(m, "        \"Cmdline\": \"");
            seq_print_json_escaped(m, own_cmdline);
            seq_printf(m, "\",\n");
        }

        seq_printf(m, "        \"VSZ_KB\": %lu,\n", vsz);
        seq_printf(m, "        \"RSS_KB\": %lu,\n", rss);
        seq_printf(m, "        \"Memory_Usage\": %lu.%lu,\n", mem_usage / 10, mem_usage % 10);
        seq_printf(m, "        \"CPU_Usage\": %lu.%lu\n", cpu_usage / 100, cpu_usage % 100);
        seq_printf(m, "     }");

        if (own_cmdline)
            kfree(own_cmdline);
    }
    rcu_read_unlock();

    seq_printf(m, "\n   ]\n}\n");
    return 0;
}

static int sysinfo_open(struct inode *inode, struct file *file)
{
    return single_open(file, sysinfo_show, NULL);
}

static const struct proc_ops sysinfo_ops = {
    .proc_open = sysinfo_open,
    .proc_read = seq_read,
};

static int __init sysinfo_init(void)
{
    proc_create(PROC_NAME, 0, NULL, &sysinfo_ops);
    printk(KERN_INFO "modulo %s cargado\n", PROC_NAME);
    return 0;
}

static void __exit sysinfo_exit(void)
{
    remove_proc_entry(PROC_NAME, NULL);
    printk(KERN_INFO "modulo %s desinstalado\n", PROC_NAME);
}

module_init(sysinfo_init);
module_exit(sysinfo_exit);