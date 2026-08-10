# Media

Default upload flow:

```text
browser -> authenticated Go presign endpoint -> R2 presigned PUT URL
browser -------------------------------------> R2 bytes
```

Store metadata through the owning business module when needed. Do not proxy large file bodies through Railway unless the feature requires server-side transformation or inspection.
