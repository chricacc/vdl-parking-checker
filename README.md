# Luxembourg's parking places status checker
Luxembourg is a nightmare when it comes to traffic. This is a simple and efficient solution to send notifications via web hooks when some parking places suddenly become full.

## build instructions
```
go build -o vdl-parking-checker vdl-parking-checker.go
```

## run instructions
```
./vdl-parking-checker \
  -webhook "https://discord.com/api/webhooks/xxx/yyy" \
  -titles "Bouillon,Martyrs,Monterey" \
  -dataUrl "https://www.vdl.lu/fr/parking/data.json"
```

## details
The program will save a JSON report containing statuses for every monitored parking places. Notifications will be sent only if the report has changed.
