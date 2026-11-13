-- Base de datos SQLite RECIBOS
-- Tabla categorias

CREATE TABLE categorias (
    codigo INTEGER PRIMARY KEY AUTOINCREMENT,
    descripcion VARCHAR(50)
);

-- Datos categorias
INSERT INTO categorias (codigo, descripcion) VALUES
(1, 'Expensas Comunes'),
(2, 'Servicios'),
(3, 'Expensas Extraordinarias'),
(4, 'Deudas');

-- Tabla clientes
CREATE TABLE clientes (
    codigo INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre VARCHAR(50),
    telefono VARCHAR(20),
    mail VARCHAR(70),
    direccion VARCHAR(70)
);

-- Datos clientes (primeros 20 registros como ejemplo)
INSERT INTO clientes (codigo, nombre, telefono, mail, direccion) VALUES
(1, 'ILIAL CONSULTORIO', '', '', 'DER ILIAL'),
(2, 'GRANDI', '', '', 'LOC COMERC'),
(3, 'PBC ALDERETE ', '', '', 'PB C'),
(4, 'PBD ALBERTO AGOTE', '', '', ' PB D'),
(5, '1A DONADIO ', '', '', '1A'),
(6, '1B DORIS ', '', '', '1B'),
(7, '1C DR CARAN *1C*', '', '', '1C'),
(8, '1D MONTEROS *1D*', '', '', '1D '),
(9, '1E SILVIA ELENA ARIAS *1E*', '', '', '1E'),
(10, '1G JOSEFINA DE LA PEÑA *1G*', '', '', '1G'),
(11, '2A LLORCA *2A*', '', '', '2A'),
(12, '2B TORRES *2B*', '', '', '2B'),
(13, '2C NICOLAS SANGENIS *2C*', '', '', '2C'),
(14, '2D MARIA LUIS BOSSOLACO *2D*', '', '', '2D'),
(15, '3A CYNTIA PEREZ 3A*', '', '', '3A'),
(16, '3C VILLAGRA ROMINA *3C*', '', '', '3C'),
(18, '3E AVELLANEDA *3E*', '', '', '3E'),
(19, '3F RICARDO ALIAS *3F*', '', '', '3F'),
(20, '3G CECILIA GUTIRREZ *3G*(silvana) ', '', '', '3G'),
(21, ' 4A NELA PAZ *4A*', '', '', '4A');

-- Tabla detallefactura
CREATE TABLE detallefactura (
    codigo INTEGER PRIMARY KEY AUTOINCREMENT,
    codigofactura INTEGER,
    codigocategoria INTEGER,
    detalle VARCHAR(100),
    precio REAL,
    cantidad INTEGER,
    pago VARCHAR(1)
);

-- Tabla facturas (inferida del dump)
CREATE TABLE facturas (
    codigo INTEGER PRIMARY KEY AUTOINCREMENT,
    codigo_cliente INTEGER,
    fecha DATE,
    total REAL,
    estado VARCHAR(20) DEFAULT 'pendiente'
);

-- Ejemplo de algunos registros detallefactura
INSERT INTO detallefactura (codigofactura, codigocategoria, detalle, precio, cantidad, pago) VALUES
(7, 1, NULL, 0, 1, ''),
(19, 1, NULL, 11111, 1, ''),
(23, 1, NULL, 13, 1, ''),
(65, 1, NULL, 1211.25, 1, 'T'),
(71, 1, 'PAGO TOTAL', 3222, 1, 'E'),
(96, 2, 'DEUDA ', 35000, 1, 'T');

