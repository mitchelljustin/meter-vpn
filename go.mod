module github.com/mvanderh/meter-vpn

go 1.12

require (
	github.com/gin-gonic/gin v1.3.0
	github.com/mvanderh/meter-vpn/daemon v0.0.0
	github.com/syndtr/goleveldb v1.0.0
)

replace github.com/mvanderh/meter-vpn/daemon v0.0.0 => ./daemon
