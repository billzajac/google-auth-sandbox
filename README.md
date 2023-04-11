## Playing with Google Auth


### nginx config

```
    location ~ /auth$ {
        proxy_pass http://localhost:9999;
    }
```

### go.mod

```
go mod init github.com/billzajac/google-auth-sandbox
go mod tidy
```
