Sphere2cubic
============

Конвертер равнопромежуточных панорм в кубические (equirectangular to cube faces converter).

На выходе - 6 квадратных jpg-файлов, по jpeg'у на каждую грань.

```
$ ./pano_sphere2cubic -help
Usage of ./pano_sphere2cubic:
  -prefix="cube_": name prefix for output images (<prefix><sidename>.jpg, cube_north.jpg, cube_top.jpg etc)
  -quality=92: output JPEG quality, 1-100
  -rot=0: additional rotation around vertical axis (degrees)
  -sides="north,south,west,east,top,bottom": sides names for output images
  -src="sphere.jpg": source file path
  -width=256: cube side width (in pixels)
```
