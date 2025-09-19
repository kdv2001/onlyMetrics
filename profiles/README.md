
получение дифа между профилями 
``
go tool pprof -top -diff_base=profiles/clientOld.out profiles/clientNew.out 
``
сбор профиля
``
curl -s http://127.0.0.1:8080/debug/pprof/profile > ./profiles/server.out   
``
веб ui + сбор
``
go tool pprof -http=":8082" -seconds=60 http://localhost:9001/debug/pprof/heap
``