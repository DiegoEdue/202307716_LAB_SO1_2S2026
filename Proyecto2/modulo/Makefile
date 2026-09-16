# Definición de módulo de kernel objeto que se va a construir
obj-m += continfo_pr2_so1_202307716.o

# Infraestructura para compilar el módulo de kernel
KDIR ?= /lib/modules/$(shell uname -r)/build
PWD := $(shell pwd)

# Compilación del módulo utilizando
all:
	$(MAKE) -C $(KDIR) M=$(PWD) modules

# Limpieza de archivos generados por la compilación
clean:
	$(MAKE) -C $(KDIR) M=$(PWD) clean