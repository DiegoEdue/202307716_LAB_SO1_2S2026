# Proyecto 1 - Sistemas Operativos 1

| Nombre | Carnet |
|---| ---|
| Diego Eduardo España Vásquez | 202307716 |

## 1. Instalación de Virt-Manager

Instalación de Virt-Manger para administrar las VMs de manera sencilla mediante una interfaz gráfica.

```bash
# Instalación de paquetes
sudo pacman -S qemu-full virt-manager libvirt dmidecode openbsd-netcat
```

- ``qemu-full``: Emulador y motor de virtualización.
- ``virt-manager``: Interfaz gráfica para gestionar VMs.
- ``libvirt``: Herramienta para controlar VMs en segundo plano.
- ``dmidecode``: Lector de datos de la BIOs.
- ``openbsd-netcat``: Comunicación entre virt-manager y libvirt.

```bash
# Habilitar servicio de libvirtd
sudo systemctl enable --now libvirtd

# Verificar que esté corriendolibvirtd
sudo systemctl status libvirtd

# Agregar usuario al grupo de libvirt para no usar sudo
sudo usermod -aG libvirt $USER

# Activar red NAT
sudo virsh net-start default

# Configurar red NAT para que se reinicie automáticamente
sudo virsh net-autostart default

# Permitir tráfico en la interfaz de la red por defecto de KVM
sudo ufw route allow in on virbr0
sudo ufw allow in on virbr0 to any port 67 proto udp
sudo ufw allow in on virbr0 to any port 53

# Confirmar subred que utiliza
virsh net-dumpxml default | grep -A2 '<ip'
```

## 2. Creación de VMs

Luego de instalar virt-manager, hay que descargar la ISO de Ubuntu Server 24.04.4. Luego para crear las máquinas virtuales fueron necesarios los siquientes pasos:

- **Crear una nueva VM:** Haz clic en el ícono de "Monitor con estrella", seleccionar "Medio de instalación local" y presionar botón de "Siguiente".
- **Elegir ISO:** Seleccionar la ISO de la ubicación donde la hayamos descargado.
- **Elegir memoria y CPU:** Para este proyecto es necesario 2048 de memoria para cada VM y 2 CPUs.
- **Espacio en disco:** Otorgar 10GB para cada VM.
- **Nombre:** Ponerle nombre a cada VM, en este caso el nombre seleccionado fue VM seguido del número al que corresponde cada máquina virtual.

## 3. Ping entre VM

En cada una de la VM es necesario instalar los paquetes para poder hacer ping entre ellas.

```bash
# Actualizar paquetes del sistema
sudo apt update

# Instalar paquetes de ping
sudo apt install iputils-ping

# Hacemos ping con la ip de red para verificar su instalación
ping -c 5 192.168.122.1
```

### 3.1 Ping VM1

```bash
# Ping a VM2
ping -c 5 192.168.122.12

# Ping a VM3
ping -c 5 192.168.122.13
```

### 3.2 Ping VM2

```bash
# Ping a VM1
ping -c 5 192.168.122.11

# Ping a VM3
ping -c 5 192.168.122.13
```

### 3.3 Ping VM3

```bash
# Ping a VM1
ping -c 5 192.168.122.11

# Ping a VM2
ping -c 5 192.168.122.12
```

## 4. Instalación de Containerd, Nerdctl y CNI en VM1

Para instalar Containerd seguí los pasos del repositorio brindado por el auxiliar [Containerd](https://github.com/JoseLorenzana272/sopes1_ejemplos/tree/main/Semana2/proyecto1-ejemplo).

```bash
# Actualizar paquetes del sistema
sudo apt-get update

# Instalar paquete oficial de containerd
sudo apt-get install -y containerd

# Verificar versión instalada
containerd --version
```

Luego de tener instalado containerd instalé nerdctl con los comandos del blog del ingeniero [Nerdctl_CNI](https://dev.to/sergioarmgpl/running-containerd-on-arm-575c).

```bash
# Actualizar paquetes del sistema
sudo apt-get update

# Instalar pversión full nerdctl que incluye las dependencias necesarias como CNI
wget https://github.com/containerd/nerdctl/releases/download/v2.0.3/nerdctl-full-2.0.3-linux-amd64.tar.gz

# Extracción de archivo en /usr/local
sudo tar -C /usr/local -xzvf nerdctl-full-2.0.3-linux-amd64.tar.gz

# Verificar versión instalada
nerdctl --version

# Primer conteneder de NGINX en el puerto 8080
sudo nerdctl run --restart always -d -p 8080:80 --name=nginx nginx

# Probar contenedor
curl localhost:8080

# Eliminar contenedor
sudo nerdctl rm -f nginx
```

## 5. Instalación de Podman en VM2

Para instalar podman se hizo de la manera tradicional en Ubuntu.

```bash
# Actualizar paquetes del sistema
sudo apt update

# Instalar Podman
sudo apt install -y podman

# Verificar versión instalada
podman --version

# Primer contenedor de NGINX en el puerto 8080
podman run -d --name nginx -p 8080:80 nginx

# Eliminar contenedor
podman rm nginx
```

## 6. Instalación de Docker en VM3

Para que la instalación fuera más sencilla utilicé el script de instalación de la página oficial de Docker, el cual se encargó de instalar todas las dependencias necesarias y configurar paquetes del sistema para que Docker funcione correctamente.

```bash
# Comando para instalación automática
curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh

# Verificar versión instalada
docker --version

# Levantar primer contenedor
sudo docker run hello-world

# Eliminar contenedor
sudo docker rm hello-world
```

Para no ejecutar todos los comandos de docker con "sudo" antes de cada comando fue necesario hacer lo siguiente.

```bash
# Agregar usuario al grupo docker
sudo adduser diegovm3 docker

# Reiniciar paquetes del sistema
sudo reboot
```
