# Revision manual - importacion personas v2.1

> ⚠️ **Registro histórico**: este cruce se hizo sobre el CSV durante la migración inicial. Hoy la fuente de verdad es PostgreSQL (`recibosgo-db`); ni este CSV ni los `*.db` se usan para corregir datos.

Pendientes detectados en dptos_dueños_inquilinos.csv y su cruce con `departamento`.

## Unidades nuevas (crear en `departamento` antes de vincular)

| unidad | numero sugerido | propietario |
|---|---|---|
| 5C | 29 | FABIAN ANGELILLI |
| 5D | 30 | FABIAN ANGELILLI |
| 6C | 35 | FABIAN ANGELILLI |
| 6D | 36 | FABIAN ANGELILLI |
| 7B | 40 | FABIAN ANGELILLI |
| 7C | 41 | FABIAN ANGELILLI |
| 7D | 42 | FABIAN ANGELILLI |
| 8C | 48 | FABIAN ANGELILLI |
| 8D | 49 | FABIAN ANGELILLI |
| 10E | 59 | GREY MEDINA |
| 13B | 74 | FABIAN ANGELILLI |
| 14B | 78 | FABIAN ANGELILLI |

## Conflictos y pendientes por unidad

- PB.CONS [1]: Local consultorio; DB propietario legacy 'ILIAL CONSULTORIO' vs CSV 'OSCAR ILAL' | INQUILINO: Nombre generico (comercio/consultorio); confirmar persona juridica o fisica
- PB.LOC [2]: Local comercial; DB 'GRANDI' coincide | INQUILINO: Solo apellido en 'inquilinos nuevos'
- PB C [3]: INQUILINO: Solo apellido en 'inquilinos nuevos'
- PB D [4]: DB 'PBD ALBERTO AGOTE' vs CSV 'AGOTE PABLO' (misma persona)
- 1B [6]: DB '1B DORIS' vs CSV 'DORA GONZALES' (misma persona) | INQUILINO: Falta apellido
- 1C [7]: INQUILINO: CSV 'inquilinos nuevos' dice 'marivel' (typo)
- 1D [8]: DB 'MONTEROS' vs CSV 'MARIO MONTERO'
- 1E [9]: CONFLICTO: DB 'SILVIA ELENA ARIAS' vs CSV 'DORA GONZALES' (tambien duena de 1B)
- 2B [12]: INQUILINO: Solo apellido en 'inquilinos nuevos'
- 2D [14]: DB 'BOSSOLACO' vs CSV 'BOSSOLASCO' | INQUILINO: Falta apellido
- 2F [113]: CONFLICTO: DB '2F ROMERO CLAUDIA' vs CSV 'ZAMORANO NORMA'
- 2G [119]: CONFLICTO: DB '2G ( ROMERO CLAUDIA' vs CSV 'NATALIA'
- 3A [15]: DB 'CYNTIA' vs CSV 'cintia' (typo) | INQUILINO: CSV: 'fransisco' (typo de FRANCISCO?) | TELEFONO propietario sospechoso: 387363603 (9 digitos)
- 3D [112]: DUPLICADO: existe tambien numero 116 '3D DIEGO ARGUELLE' (id 94); confirmar cual es la unidad valida | INQUILINO: falta apellido (nombre unico): NICOLAS
- 3G [20]: DB 'GUTIRREZ' vs CSV 'GUTIERRE' (typo) | INQUILINO: falta apellido (nombre unico): SILVANA
- 4A [21]: INQUILINO: falta apellido (nombre unico): MARIANA
- 4B [22]: DB 'FORTINI' vs CSV 'FORTINO' | INQUILINO: Solo apellido en 'inquilinos nuevos'
- 4E [24]: INQUILINO: falta apellido (nombre unico): AGUSTINA
- 4F [25]: INQUILINO: falta apellido (nombre unico): FLORENCIA
- 4G [26]: CSV: INQUILINO 'PROP.' pero telefono/columna 'inquilinos nuevos' indican inquilino 'herrera' | INQUILINO: CSV INQUILINO='PROP.' pero hay telefono y 'herrera' en col. inquilinos nuevos
- 5A [27]: INQUILINO: falta apellido (nombre unico): LUCY
- 5C [29]: NUEVO departamento (falta en la DB); numero sugerido 29 (hueco tras 5B=28)
- 5D [30]: NUEVO departamento (falta en la DB); numero sugerido 30
- 5E [31]: DB 'SUAREZ' vs CSV 'SUARES'
- 5F [32]: DB 'PEREZ FRARESSCO' vs CSV 'PERES FRERESSO'
- 6A [124]: DB '6A. AGOSTINA' (refiere a inquilino); propietario CSV 'ANIBAL IBARRA'
- 6B [34]: DB 'SRA DE AGUILAR' vs CSV 'MIRTA DE AGUILAR'
- 6C [35]: NUEVO departamento (falta en la DB); numero sugerido 35
- 6D [36]: NUEVO departamento (falta en la DB); numero sugerido 36
- 6E+ [37]: DB lo registra como '6F PADILLA' (id 32); confirmar si la unidad es 6E o 6F | INQUILINO: CSV 'inquilinos nuevos' dice 'mariana padilla' (conflicto con GODOY)
- 6G+ [38]: DB '6G SERRANO DIAS ROCIO' coincide con CSV 'DIAZ VALENZUELA / ROCIO DIAS'
- 7A [39]: DB 'GUSTABO' vs CSV 'GUSTAVO' (typo) | INQUILINO: falta apellido (nombre unico): ESTEFANIA
- 7B [40]: NUEVO departamento (falta en la DB); numero sugerido 40
- 7C [41]: NUEVO departamento (falta en la DB); numero sugerido 41
- 7D [42]: NUEVO departamento (falta en la DB); numero sugerido 42
- 7E [90]: INQUILINO: falta apellido (nombre unico): ALEJANDRO
- 7F [94]: INQUILINO: Solo apellido en 'inquilinos nuevos'
- 7G [117]: INQUILINO: Solo apellido en 'inquilinos nuevos'
- 8A [46]: CSV: inquilino 'FATIMA BLANCO' vs 'inquilinos nuevos' = 'Leito' | INQUILINO: CSV 'inquilinos nuevos' dice 'Leito' (apellido alternativo?)
- 8B [47]: CSV: inquilino 'CHELO' vs 'inquilinos nuevos' = 'Facundo' | INQUILINO: CSV 'inquilinos nuevos' dice 'Facundo' (conflicto; confirmar)
- 8C [48]: NUEVO departamento (falta en la DB); numero sugerido 48 (hueco 8A=46,8B=47,8E=50)
- 8D [49]: NUEVO departamento (falta en la DB); numero sugerido 49
- 8E [50]: CSV repite el nombre del propietario en columna inquilino; probablemente NO hay inquilino
- 8F [120]: DB 'ZAVADISKA' vs CSV 'ZAVADIKA'
- 9C [52]: DB 'MUTAFAF' vs CSV 'MUTAFA'
- 9E [89]: DB '9E BUNADER ( INQUILINO VERA' coincide con inquilino 'JAZMIN VERA'
- 9F [97]: DB '9f bunader ( PABLO SORIA' vs CSV inquilino 'KARINA UGARTE' + 'Soria' | INQUILINO: CSV 'inquilinos nuevos' dice 'Soria'; DB legacy dice 'PABLO SORIA' (conflicto)
- 9G [83]: INQUILINO: Solo apellido en 'inquilinos nuevos'
- 10A [56]: DB 'GUTIERRES' vs CSV 'GUTIERRE'; CSV inquilino 'BENJAMIN CORDOBA' vs 'Helena' | INQUILINO: CSV 'inquilinos nuevos' dice 'Helena' (conflicto) | TELEFONO inquilino sospechoso: 38155751105 (11 digitos)
- 10B [57]: CONFLICTO: DB '10B CASTILLO' vs CSV 'GUSTAVO CARTILLO' | TELEFONO propietario sospechoso: 9176251905 (10 digitos)
- 10C [58]: DB 'CATALINA' vs CSV 'CATALINA GIRI'
- 10E [59]: NUEVO departamento (falta en la DB); numero sugerido 59 (hueco 10C=58,10F=60)
- 10F [60]: CSV inquilino 'WALTER' vs 'inquilinos nuevos' = 'Viviana' (repite propietario) | INQUILINO: CSV 'inquilinos nuevos' dice 'Viviana' = nombre del propietario (conflicto)
- 11A [62]: CONFLICTO: DB '11A DOSMIK ROSA' vs CSV 'JOSE GERBAN' | INQUILINO: Solo apellido en 'inquilinos nuevos' | TELEFONO inquilino sospechoso: 38144000902 (11 digitos)
- 11B [63]: CONFLICTO: DB '11B GETTAR' vs CSV 'CECILIA ROMERO'
- 11C [118]: DB '11C BUNADER FACUNDO' vs CSV inquilino 'Bichara' | INQUILINO: Solo apellido en 'inquilinos nuevos'
- 11G [67]: CSV inquilino 'MARIA JOSE' + 'Cabrera Maria' (orden invertido) | INQUILINO: CSV 'inquilinos nuevos' dice 'Cabrera Maria' (orden invertido)
- 12A [68]: DB 'BARRERA JORGE' vs CSV 'BARRERA'
- 12F [71]: DB '(CHAIN' vs CSV 'LEONEL CHAING'
- 12G [72]: DB '(TVANINA' vs CSV 'VANINA TAPIA'
- 13A [73]: CSV inquilino 'MATIAS CARRERA' + 'Molina' | INQUILINO: CSV 'inquilinos nuevos' dice 'Molina' (apellido alternativo?)
- 13B [74]: NUEVO departamento (falta en la DB); numero sugerido 74 (hueco 13A=73,13C=75)
- 13C [75]: DB 'CAMBERA VICTOR' vs CSV 'VICTOR CAMBERA'
- 13G [76]: Propietario incluye alias '(SABRINA)'; se normaliza a 'VANINA MEDINA NUÑEZ' | PROPIETARIO: Propietario incluye alias '(SABRINA)'; se normaliza
- 14A [77]: DB '14A VERO ANGELILLI *14A* (MANSILLA' coincide con inquilino 'ALEJANDRO MANSILLA'
- 14B [78]: NUEVO departamento (falta en la DB); numero sugerido 78 (hueco 14A=77,14E=79)
- 14F [80]: CONFLICTO: DB '14F IMELDA RICHARD' vs CSV 'ALVARO FERRARI'
- COLISION: 'ALARCON' aparece como inquilino y propietario (posible persona distinta con mismo nombre incompleto)

Total de personas: 123 | Vinculos: 148 | Unidades nuevas: 12
