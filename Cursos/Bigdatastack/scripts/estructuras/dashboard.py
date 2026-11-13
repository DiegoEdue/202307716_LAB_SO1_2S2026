ventas_por_dia = {
    "lunes": 1560.80,
    "martes": 980.50,
    "miércoles": 1430.20,
    "jueves": 1250.00,
    "viernes": 890.00,
}

# Construye el reporte
# Pista para mejor_dia/peor_dia: max(ventas_por_dia, key=ventas_por_dia.get) devuelve
# la CLAVE cuyo valor es el más alto. El "key=" le dice a max con qué criterio comparar:
# en vez de comparar los nombres de los días, compara lo que vendió cada uno.
# Para el peor, lo mismo con min(...).
# Y un aviso de forma, no de fondo: al cambiar el valor de cada línea, deja
# la COMA del final. Si se te va, Python avisa muy lejos de donde está el
# fallo («forgot a comma?» o «closing parenthesis '}' does not match»).
reporte = {
    "total_ventas": sum(ventas_por_dia.values()),      # <- suma de todos los días. Pista: .values()
    "num_dias": len(ventas_por_dia),            # <- cuántos días hay
    "promedio_diario": sum(ventas_por_dia.values())/len(ventas_por_dia),   # <- total dividido entre los días
    "mejor_dia": max(ventas_por_dia, key=ventas_por_dia.get),         # <- el día que más vendió
    "peor_dia": min(ventas_por_dia, key=ventas_por_dia.get),          # <- el día que menos vendió
}

# No toques estas líneas: son las que muestran el reporte
print("REPORTE SEMANAL")
for clave, valor in reporte.items():
    print(f"  {clave}: {valor}")