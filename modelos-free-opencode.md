# Modelos Free de OpenCode

Listado de modelos de chat con costo **$0** (input y output) disponibles en el proveedor `opencode`, obtenidos de `opencode/models`.

> **Nota:** OpenCode no expone un campo de "especialidad" por modelo. Las especialidades que se indican a continuación están inferidas del nombre/familia del modelo y del conocimiento general de cada proveedor; úsalas como referencia, no como garantía de comportamiento.

## Modelos gratuitos (costo 0 / 0)

| # | Modelo | ID | Especialidad (inferida) | Variantes | Estado |
|---|--------|----|-------------------------|-----------|--------|
| 1 | Flint 40B Free | `opencode/flint-40b-free` | Generalista de propósito medio (40B parámetros), equilibrado entre calidad y velocidad | low, medium, high, xhigh, max | active |
| 2 | Nemotron 3.5 Super VL Free | `opencode/nemotron-3.5-super-vl-free` | Modelo visión-lenguaje (VL): razonamiento general con capacidad de procesar imágenes | none, minimal, low, medium, high, xhigh | active |
| 3 | Beam Free | `opencode/beam-free` | Generalista ligero, orientado a respuestas rápidas y tareas cotidianas | low, medium, high, xhigh, max | active |
| 4 | MiMo-V2.6-Flash Free | `opencode/mimo-v2.6-flash-free` | Modelo "Flash": inferencia de baja latencia para consultas simples y rápidas | — | active |
| 5 | DeepSeek V4.1 Flash Free | `opencode/deepseek-v4.1-flash-free` | Familia DeepSeek: fuerte en razonamiento, código y matemáticas; versión Flash para velocidad | low, high, max | active |
| 6 | GLM-5.3-Flash Free | `opencode/glm-5.3-flash-free` | Modelo generalista (familia GLM de Zhipu AI) optimizado para respuestas rápidas | low, high, max | active |
| 7 | Big Pickle | `opencode/big-pickle` | Modelo de nicho / propósito especial sin variantes expuestas; uso experimental | — | active |

## Modelo actual de esta sesión

Este agente está ejecutándose sobre **Nemotron 3.5 Super VL Free** (`opencode/nemotron-3.5-super-vl-free`), un modelo visión-lenguaje gratuito y activo.

## Cómo consultar la lista actualizada

Desde OpenCode (Code Mode), ejecutar:

```js
const models = await tools.opencode.models({ provider: "opencode", type: "chat" });
const free = models.providers[0].models.filter(m =>
  m.cost.some(c => c.input === 0 && c.output === 0)
);
```

Los precios se expresan en USD por millón de tokens.
