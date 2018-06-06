export IMAGE := introllo/actuary:actuary_image
export TLS_KEY := ./domain.key
export TLS_CERT :=  ./domain.crt
export TOKEN_PASSWORD := ./token_password.txt

export HTTP_PROXY := http://proxy-chain.intel.com:911
export HTTPS_PROXY := http://proxy-chain.intel.com:912

$(shell bash generate_certs.sh)

IP_ADDRESS := $(shell bash ip_address.sh)

default: setup  
	docker stack deploy -c docker-compose.yml actuary
	@echo "Use address below to view results:"
	@echo "$(IP_ADDRESS)"

setup:
	docker build . --tag "$(IMAGE)"
	docker push "$(IMAGE)"

clean:
	docker stack rm actuary

quick-build: 
	go build -o actuaryBinary github.com/introllo/actuary/cmd/actuary
	./actuaryBinary server
