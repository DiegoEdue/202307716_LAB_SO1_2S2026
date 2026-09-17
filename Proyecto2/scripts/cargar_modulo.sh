#!/bin/bash

set -e

cd "$(dirname "$0")/.."

echo "Compilando modulo kernel..."
make -C modulo

echo "Cargando modulo..."
lsmod | grep -q continfo_pr2_so1_202307716 || sudo insmod modulo/continfo_pr2_so1_202307716.ko

echo "Modulo listo. Verificando /proc/continfo_pr2_so1_202307716"
ls -la /proc/continfo_pr2_so1_202307716 || echo "El archivo proc no esta disponible"