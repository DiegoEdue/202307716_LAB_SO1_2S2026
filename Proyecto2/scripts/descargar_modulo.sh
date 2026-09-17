#!/bin/bash

set -e

echo "Descargando modulo kernel..."
sudo rmmod continfo_pr2_so1_202307716 2>/dev/null || echo "Modulo no estaba cargado"

echo "Modulo descargado. Archivo /proc/continfo_pr2_so1_202307716 no estara disponible."