test:
	go test -v -race

cover:
	rm -rf *.coverprofile
	go test -coverprofile=ratelimit.coverprofile
	gover
	go tool cover -html=ratelimit.coverprofile