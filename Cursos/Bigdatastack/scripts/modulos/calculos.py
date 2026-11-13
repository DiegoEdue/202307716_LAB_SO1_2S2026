# --- archivo: calculos.py ---
"""Módulo con funciones de cálculo para datos de ventas."""

def calcular_total(cantidad, precio_unitario):
    """Calcula el total de una línea de venta."""
    return cantidad * precio_unitario

def aplicar_iva(importe, tasa=0.21):
    """Aplica IVA a un importe."""
    return importe * (1 + tasa)

def calcular_descuento(importe, porcentaje):
    """Aplica un descuento porcentual."""
    return importe * (1 - porcentaje / 100)