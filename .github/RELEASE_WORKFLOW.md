# Workflow de Releases Automáticos

Este proyecto utiliza GitHub Actions para crear releases automáticamente cuando se hace un commit que comienza con el símbolo `+`.

## Cómo funciona

1. **Trigger**: El workflow se activa automáticamente cuando haces push a la rama `main` con un commit que empiece con `+`

2. **Versionado**: El sistema lee el último tag de release y automáticamente incrementa la versión
   - Si no hay releases previos: crea `v0.1`
   - Si el último release es `v0.1`: crea `v0.2`
   - Si el último release es `v0.2`: crea `v0.3`
   - Si el último release es `v0.9`: crea `v0.10`
   - Y así sucesivamente...

3. **Notas del Release**: El contenido del commit (sin el prefijo `+`) se usa como descripción del release

## Ejemplos de uso

### Crear un nuevo release

```bash
git add .
git commit -m "+ Se agrega nueva funcionalidad de autenticación"
git push origin main
```

Esto creará automáticamente:
- Un nuevo tag (ej: `v0.2`)
- Un release en GitHub con el título "Release v0.2"
- Descripción: "Se agrega nueva funcionalidad de autenticación"

### Commits normales (sin crear release)

```bash
git add .
git commit -m "Fix: corrección de bug menor"
git push origin main
```

Este commit NO creará un release porque no empieza con `+`

## Requisitos

- El workflow requiere permisos de escritura en el repositorio
- Solo funciona en la rama `main`
- El formato del commit debe ser: `+ [descripción]`

## Verificar releases

Puedes ver todos los releases creados en:
`https://github.com/[tu-usuario]/[tu-repo]/releases`

## Notas técnicas

- El workflow usa `bc` para cálculos decimales precisos
- Los tags siguen el formato `vX.Y` (ej: v0.1, v0.2, v0.3)
- El workflow solo se ejecuta si el commit message empieza exactamente con `+`
