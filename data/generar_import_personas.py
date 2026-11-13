#!/usr/bin/env python3
"""Genera CSVs de importacion para las tablas v2.1 `persona` y `departamento_persona`
a partir de dptos_dueños_inquilinos.csv y la tabla `departamento` actual (recibos.db).

Salidas (en data/):
  personas_import.csv          -> filas para `persona`
  vinculos_import.csv          -> filas para `departamento_persona`
  departamentos_nuevos.csv     -> unidades en el CSV que faltan en `departamento`
  REVISION_MANUAL.md           -> conflictos, datos incompletos y pendientes
"""

import csv
import io
import re
import sqlite3
import unicodedata
from collections import OrderedDict

BASE = "/home/daniel/Proyectos/recibosGo"
SRC_CSV = f"{BASE}/dptos_dueños_inquilinos.csv"
DB = f"{BASE}/recibos.db"
OUT = f"{BASE}/data"

# ---------------------------------------------------------------------------
# Mapping CSV DEPTO -> departamento (id, numero) en la DB actual.
# id/numero = None => la unidad no existe aun en `departamento` (crear).
# flag        => observaciones de integracion / conflicto con datos de la DB.
# ---------------------------------------------------------------------------
MAPPING = {
    "PB.CONS": (1, "1", "Local consultorio; DB propietario legacy 'ILIAL CONSULTORIO' vs CSV 'OSCAR ILAL'"),
    "PB.LOC": (2, "2", "Local comercial; DB 'GRANDI' coincide"),
    "PB C": (3, "3", ""),
    "PB D": (4, "4", "DB 'PBD ALBERTO AGOTE' vs CSV 'AGOTE PABLO' (misma persona)"),
    "1A": (5, "5", ""),
    "1B": (6, "6", "DB '1B DORIS' vs CSV 'DORA GONZALES' (misma persona)"),
    "1C": (7, "7", ""),
    "1D": (8, "8", "DB 'MONTEROS' vs CSV 'MARIO MONTERO'"),
    "1E": (9, "9", "CONFLICTO: DB 'SILVIA ELENA ARIAS' vs CSV 'DORA GONZALES' (tambien duena de 1B)"),
    "1F": (80, "101", ""),
    "1G": (10, "10", ""),
    "2A": (11, "11", ""),
    "2B": (12, "12", ""),
    "2C": (13, "13", ""),
    "2D": (14, "14", "DB 'BOSSOLACO' vs CSV 'BOSSOLASCO'"),
    "2E": (64, "85", ""),
    "2F": (91, "113", "CONFLICTO: DB '2F ROMERO CLAUDIA' vs CSV 'ZAMORANO NORMA'"),
    "2G": (97, "119", "CONFLICTO: DB '2G ( ROMERO CLAUDIA' vs CSV 'NATALIA'"),
    "3A": (15, "15", "DB 'CYNTIA' vs CSV 'cintia' (typo)"),
    "3C": (16, "16", ""),
    "3D": (90, "112", "DUPLICADO: existe tambien numero 116 '3D DIEGO ARGUELLE' (id 94); confirmar cual es la unidad valida"),
    "3E": (17, "18", ""),
    "3F": (18, "19", ""),
    "3G": (19, "20", "DB 'GUTIRREZ' vs CSV 'GUTIERRE' (typo)"),
    "4A": (20, "21", ""),
    "4B": (21, "22", "DB 'FORTINI' vs CSV 'FORTINO'"),
    "4C": (65, "86", ""),
    "4D": (22, "23", ""),
    "4E": (23, "24", ""),
    "4F": (24, "25", ""),
    "4G": (25, "26", "CSV: INQUILINO 'PROP.' pero telefono/columna 'inquilinos nuevos' indican inquilino 'herrera'"),
    "5A": (26, "27", ""),
    "5B": (27, "28", ""),
    "5C": (None, "29", "NUEVO departamento (falta en la DB); numero sugerido 29 (hueco tras 5B=28)"),
    "5D": (None, "30", "NUEVO departamento (falta en la DB); numero sugerido 30"),
    "5E": (28, "31", "DB 'SUAREZ' vs CSV 'SUARES'"),
    "5F": (29, "32", "DB 'PEREZ FRARESSCO' vs CSV 'PERES FRERESSO'"),
    "5G": (30, "33", ""),
    "6A": (102, "124", "DB '6A. AGOSTINA' (refiere a inquilino); propietario CSV 'ANIBAL IBARRA'"),
    "6B": (31, "34", "DB 'SRA DE AGUILAR' vs CSV 'MIRTA DE AGUILAR'"),
    "6C": (None, "35", "NUEVO departamento (falta en la DB); numero sugerido 35"),
    "6D": (None, "36", "NUEVO departamento (falta en la DB); numero sugerido 36"),
    "6E+": (32, "37", "DB lo registra como '6F PADILLA' (id 32); confirmar si la unidad es 6E o 6F"),
    "6G+": (33, "38", "DB '6G SERRANO DIAS ROCIO' coincide con CSV 'DIAZ VALENZUELA / ROCIO DIAS'"),
    "7A": (34, "39", "DB 'GUSTABO' vs CSV 'GUSTAVO' (typo)"),
    "7B": (None, "40", "NUEVO departamento (falta en la DB); numero sugerido 40"),
    "7C": (None, "41", "NUEVO departamento (falta en la DB); numero sugerido 41"),
    "7D": (None, "42", "NUEVO departamento (falta en la DB); numero sugerido 42"),
    "7E": (69, "90", ""),
    "7F": (73, "94", ""),
    "7G": (95, "117", ""),
    "8A": (35, "46", "CSV: inquilino 'FATIMA BLANCO' vs 'inquilinos nuevos' = 'Leito'"),
    "8B": (36, "47", "CSV: inquilino 'CHELO' vs 'inquilinos nuevos' = 'Facundo'"),
    "8C": (None, "48", "NUEVO departamento (falta en la DB); numero sugerido 48 (hueco 8A=46,8B=47,8E=50)"),
    "8D": (None, "49", "NUEVO departamento (falta en la DB); numero sugerido 49"),
    "8E": (37, "50", "CSV repite el nombre del propietario en columna inquilino; probablemente NO hay inquilino"),
    "8F": (98, "120", "DB 'ZAVADISKA' vs CSV 'ZAVADIKA'"),
    "8G": (38, "51", ""),
    "9C": (39, "52", "DB 'MUTAFAF' vs CSV 'MUTAFA'"),
    "9E": (68, "89", "DB '9E BUNADER ( INQUILINO VERA' coincide con inquilino 'JAZMIN VERA'"),
    "9F": (76, "97", "DB '9f bunader ( PABLO SORIA' vs CSV inquilino 'KARINA UGARTE' + 'Soria'"),
    "9G": (63, "83", ""),
    "10A": (40, "56", "DB 'GUTIERRES' vs CSV 'GUTIERRE'; CSV inquilino 'BENJAMIN CORDOBA' vs 'Helena'"),
    "10B": (41, "57", "CONFLICTO: DB '10B CASTILLO' vs CSV 'GUSTAVO CARTILLO'"),
    "10C": (42, "58", "DB 'CATALINA' vs CSV 'CATALINA GIRI'"),
    "10E": (None, "59", "NUEVO departamento (falta en la DB); numero sugerido 59 (hueco 10C=58,10F=60)"),
    "10F": (43, "60", "CSV inquilino 'WALTER' vs 'inquilinos nuevos' = 'Viviana' (repite propietario)"),
    "10G": (44, "61", ""),
    "11A": (45, "62", "CONFLICTO: DB '11A DOSMIK ROSA' vs CSV 'JOSE GERBAN'"),
    "11B": (46, "63", "CONFLICTO: DB '11B GETTAR' vs CSV 'CECILIA ROMERO'"),
    "11C": (96, "118", "DB '11C BUNADER FACUNDO' vs CSV inquilino 'Bichara'"),
    "11E": (47, "65", ""),
    "11F": (48, "66", ""),
    "11G": (49, "67", "CSV inquilino 'MARIA JOSE' + 'Cabrera Maria' (orden invertido)"),
    "12A": (50, "68", "DB 'BARRERA JORGE' vs CSV 'BARRERA'"),
    "12B": (51, "69", ""),
    "12C": (52, "70", ""),
    "12F": (53, "71", "DB '(CHAIN' vs CSV 'LEONEL CHAING'"),
    "12G": (54, "72", "DB '(TVANINA' vs CSV 'VANINA TAPIA'"),
    "13A": (55, "73", "CSV inquilino 'MATIAS CARRERA' + 'Molina'"),
    "13B": (None, "74", "NUEVO departamento (falta en la DB); numero sugerido 74 (hueco 13A=73,13C=75)"),
    "13C": (56, "75", "DB 'CAMBERA VICTOR' vs CSV 'VICTOR CAMBERA'"),
    "13G": (57, "76", "Propietario incluye alias '(SABRINA)'; se normaliza a 'VANINA MEDINA NUÑEZ'"),
    "14A": (58, "77", "DB '14A VERO ANGELILLI *14A* (MANSILLA' coincide con inquilino 'ALEJANDRO MANSILLA'"),
    "14B": (None, "78", "NUEVO departamento (falta en la DB); numero sugerido 78 (hueco 14A=77,14E=79)"),
    "14E": (59, "79", ""),
    "14F": (60, "80", "CONFLICTO: DB '14F IMELDA RICHARD' vs CSV 'ALVARO FERRARI'"),
    "14G": (61, "81", ""),
}

# Sobrescrituras de nombre final para inquilinos ambiguos:
# depto -> (nombre_inquilino_final, flag_adicional o None)
TENANT_OVERRIDES = {
    "PB.CONS": ("CONSULTORIO", "Nombre generico (comercio/consultorio); confirmar persona juridica o fisica"),
    "1C": ("MARIBEL", "CSV 'inquilinos nuevos' dice 'marivel' (typo)"),
    "4G": ("HERRERA", "CSV INQUILINO='PROP.' pero hay telefono y 'herrera' en col. inquilinos nuevos"),
    "8A": ("FATIMA BLANCO", "CSV 'inquilinos nuevos' dice 'Leito' (apellido alternativo?)"),
    "8B": ("CHELO", "CSV 'inquilinos nuevos' dice 'Facundo' (conflicto; confirmar)"),
    "8E": (None, "CSV repite nombre del propietario; probablemente sin inquilino"),
    "9F": ("KARINA UGARTE", "CSV 'inquilinos nuevos' dice 'Soria'; DB legacy dice 'PABLO SORIA' (conflicto)"),
    "10A": ("BENJAMIN CORDOBA", "CSV 'inquilinos nuevos' dice 'Helena' (conflicto)"),
    "10F": ("WALTER", "CSV 'inquilinos nuevos' dice 'Viviana' = nombre del propietario (conflicto)"),
    "11G": ("MARIA JOSE CABRERA", "CSV 'inquilinos nuevos' dice 'Cabrera Maria' (orden invertido)"),
    "13A": ("MATIAS CARRERA", "CSV 'inquilinos nuevos' dice 'Molina' (apellido alternativo?)"),
    "6E+": ("MARIANA GODOY", "CSV 'inquilinos nuevos' dice 'mariana padilla' (conflicto con GODOY)"),
    "3A": ("FRANSISCO", "CSV: 'fransisco' (typo de FRANCISCO?)"),
    "2D": ("NORMA", "Falta apellido"),
    "1B": ("FERNANDA", "Falta apellido"),
    "2G": (None, None),
}

NO_TENANT = {"PROP", "PROP.", "-", ""}

# Sobrescrituras del nombre del propietario (depto -> (nombre_final, flag))
PROP_OVERRIDES = {
    "13G": ("VANINA MEDINA NUÑEZ", "Propietario incluye alias '(SABRINA)'; se normaliza"),
}


def norm_key(name: str) -> str:
    name = unicodedata.normalize("NFD", name)
    name = "".join(c for c in name if unicodedata.category(c) != "Mn")
    return re.sub(r"\s+", " ", name).strip().upper()


def clean(name: str) -> str:
    return re.sub(r"\s+", " ", name or "").strip().upper()


def unidad_nombre(depto: str) -> str:
    u = depto.strip().upper().replace(".", " ")
    u = re.sub(r"\s+", " ", u)
    return u


def parse_rows():
    with open(SRC_CSV, newline="", encoding="utf-8-sig") as f:
        rd = csv.reader(f)
        header = next(rd)
        for row in rd:
            if not row or not row[0].strip():
                continue
            depto = row[0].strip()
            prop = clean(row[1]) if len(row) > 1 else ""
            inqui = clean(row[2]) if len(row) > 2 else ""
            tel_prop = clean(row[3]) if len(row) > 3 else ""
            tel_inq = clean(row[4]) if len(row) > 4 else ""
            nuevos = clean(row[5]) if len(row) > 5 else ""
            yield {
                "depto": depto,
                "prop": prop,
                "inqui": inqui,
                "tel_prop": tel_prop,
                "tel_inq": tel_inq,
                "nuevos": nuevos,
            }


def resolver_inquilino(r):
    depto = r["depto"]
    inq, nuevos, tel = r["inqui"], r["nuevos"], r["tel_inq"]
    prop = r["prop"]

    if depto in TENANT_OVERRIDES:
        nombre, flag = TENANT_OVERRIDES[depto]
        if not nombre:
            return None, None, flag
        return nombre, tel or None, flag

    k_new = norm_key(nuevos)
    k_prop = norm_key(prop) if prop else ""
    prop_tokens = set(k_prop.split()) if prop else set()
    # "PROP" / "PROP." / "-" => sin inquilino (el propietario habita la unidad).
    if inq in NO_TENANT:
        if (not nuevos
                or k_new in NO_TENANT
                or (prop and (k_new == k_prop or k_new in prop_tokens))):
            return None, None, None
        return nuevos, tel or None, "Solo apellido en 'inquilinos nuevos'"

    k_inq = norm_key(inq)
    if nuevos:
        if k_new == k_inq or k_new == norm_key(inq.split()[-1]):
            return inq, tel or None, None
        if len(nuevos.split()) == 1:
            return f"{inq} {nuevos}", tel or None, None
        return inq, tel or None, f"'inquilinos nuevos'='{nuevos}' no coincide con inquilino '{inq}'"
    return inq, tel or None, None


def main():
    con = sqlite3.connect(DB)
    cur = con.cursor()
    cur.execute("SELECT id, numero FROM departamento")
    db_ids = {str(n): i for i, n in cur.fetchall()}
    con.close()

    personas = OrderedDict()  # norm_key -> {tipo, nombre, telefono}
    vinculos = []
    nuevos_dptos = OrderedDict()
    revision = []

    for r in parse_rows():
        depto = r["depto"]
        if depto not in MAPPING:
            revision.append(f"{depto}: SIN MAPEO en MAPPING (revisar)")
            continue
        depto_id, numero, flag_dpt = MAPPING[depto]
        unidad = unidad_nombre(depto)
        row_flags = []
        if flag_dpt:
            row_flags.append(flag_dpt)

        # Propietario
        prop = clean(r["prop"]) or None
        if depto in PROP_OVERRIDES:
            prop, ovr = PROP_OVERRIDES[depto]
            row_flags.append(f"PROPIETARIO: {ovr}")
        if prop:
            k = norm_key(prop)
            if k not in personas:
                personas[k] = {"tipo": "propietario", "nombre": prop, "telefono": clean(r["tel_prop"]) or None}
            else:
                if not personas[k]["telefono"]:
                    personas[k]["telefono"] = clean(r["tel_prop"]) or None
            vinculos.append([depto_id or "", numero or "", unidad, prop, "propietario"])

        # Inquilino
        inq_nombre, inq_tel, inq_flag = resolver_inquilino(r)
        if inq_nombre:
            if inq_flag:
                row_flags.append(f"INQUILINO: {inq_flag}")
            elif len(inq_nombre.split()) == 1:
                row_flags.append(f"INQUILINO: falta apellido (nombre unico): {inq_nombre}")
            k = norm_key(inq_nombre)
            if k not in personas:
                personas[k] = {"tipo": "inquilino", "nombre": inq_nombre, "telefono": inq_tel}
            else:
                if not personas[k]["telefono"]:
                    personas[k]["telefono"] = inq_tel
            vinculos.append([depto_id or "", numero or "", unidad, inq_nombre, "inquilino"])

        # Telefonos sospechosos
        for label, tel in (("propietario", r["tel_prop"]), ("inquilino", r["tel_inq"])):
            if tel and (len(tel) == 9 or len(tel) == 11 or (len(tel) == 10 and tel.startswith("9"))):
                row_flags.append(f"TELEFONO {label} sospechoso: {tel} ({len(tel)} digitos)")

        # Unidades nuevas
        if depto_id is None:
            nuevos_dptos.setdefault(unidad, [numero or "", prop or "", unidad])

        if row_flags:
            revision.append(f"{depto} [{numero or 'NUEVO'}]: " + " | ".join(row_flags))

    # Colisiones: mismo nombre normalizado usado como propietario E inquilino
    roles = {}
    for v in vinculos:
        k = norm_key(v[3])
        roles.setdefault(k, set()).add(v[4])
    for k, rs in roles.items():
        if len(rs) > 1:
            revision.append(
                f"COLISION: '{k}' aparece como {' y '.join(sorted(rs))} "
                "(posible persona distinta con mismo nombre incompleto)"
            )

    # ---- Salidas ----
    with open(f"{OUT}/personas_import.csv", "w", newline="", encoding="utf-8") as f:
        w = csv.writer(f)
        w.writerow(["tipo", "nombre", "telefono", "email", "activo"])
        for p in personas.values():
            w.writerow([p["tipo"], p["nombre"], p["telefono"] or "", "", 1])

    with open(f"{OUT}/vinculos_import.csv", "w", newline="", encoding="utf-8") as f:
        w = csv.writer(f)
        w.writerow(["departamento_id", "numero", "unidad", "nombre_persona", "rol"])
        for v in vinculos:
            w.writerow(v)

    with open(f"{OUT}/departamentos_nuevos.csv", "w", newline="", encoding="utf-8") as f:
        w = csv.writer(f)
        w.writerow(["unidad", "numero_sugerido", "propietario", "inquilino"])
        for unidad, (num, prop, _u) in nuevos_dptos.items():
            inq = next((v[3] for v in vinculos if v[2] == unidad and v[4] == "inquilino"), "")
            w.writerow([unidad, num, prop, inq])

    with open(f"{OUT}/REVISION_MANUAL.md", "w", encoding="utf-8") as f:
        f.write("# Revision manual - importacion personas v2.1\n\n")
        f.write("Pendientes detectados en dptos_dueños_inquilinos.csv y su cruce con `departamento`.\n\n")
        f.write("## Unidades nuevas (crear en `departamento` antes de vincular)\n\n")
        f.write("| unidad | numero sugerido | propietario |\n|---|---|---|\n")
        for unidad, (num, prop, _u) in nuevos_dptos.items():
            f.write(f"| {unidad} | {num} | {prop} |\n")
        f.write("\n## Conflictos y pendientes por unidad\n\n")
        for line in revision:
            f.write(f"- {line}\n")
        f.write(f"\nTotal de personas: {len(personas)} | Vinculos: {len(vinculos)} | Unidades nuevas: {len(nuevos_dptos)}\n")

    print(f"personas: {len(personas)}")
    print(f"vinculos: {len(vinculos)}")
    print(f"unidades nuevas: {len(nuevos_dptos)}")
    print(f"revisiones: {len(revision)}")
    print(f"propietarios: {sum(1 for p in personas.values() if p['tipo']=='propietario')}")
    print(f"inquilinos: {sum(1 for p in personas.values() if p['tipo']=='inquilino')}")


if __name__ == "__main__":
    main()
