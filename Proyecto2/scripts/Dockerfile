# Imagen "intrusa": simula, dentro de su propio filesystem de contenedor,intentos repetidos de lectura de archivos sensibles.

FROM alpine:latest

COPY intruso.sh /usr/local/bin/intruso.sh
RUN chmod +x /usr/local/bin/intruso.sh

ENTRYPOINT ["/usr/local/bin/intruso.sh"]