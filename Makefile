.PHONY: vet
vet:
	@echo "---- Vetting ----"
	go vet ./...
	@echo "---- Successfully Vet ----\n"