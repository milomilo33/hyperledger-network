package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

type CarDefect struct {
	Description string
	RepairPrice float32
}

type Car struct {
	ID               string
	Make             string
	Model            string
	YearManufactured int
	Color            string
	OwnerID          string
	TransferPrice    float32
	PaintPrice       float32
	CarDefects       []CarDefect
}

type Person struct {
	ID        string
	Name      string
	Surname   string
	Email     string
	MoneyLeft float32
}

func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {

	initCars := []Car{
		{ID: "car1", Make: "Mercedes", Model: "S-klasse", YearManufactured: 2020, Color: "grey", OwnerID: "person1", TransferPrice: 8000, PaintPrice: 100, CarDefects: []CarDefect{
			{Description: "Punctured front left tire", RepairPrice: 35},
		}},
		{ID: "car2", Make: "Volkswagen", Model: "Buba", YearManufactured: 2009, Color: "pink", OwnerID: "person3", TransferPrice: 4500, PaintPrice: 80, CarDefects: []CarDefect{
			{Description: "Touchscreen not responding", RepairPrice: 50},
		}},
		{ID: "car3", Make: "Dacia", Model: "Sandero", YearManufactured: 2008, Color: "grey", OwnerID: "person3", TransferPrice: 3400, PaintPrice: 70, CarDefects: []CarDefect{
			{Description: "Wiper engine broken", RepairPrice: 200},
			{Description: "Radio spontaneously turns off", RepairPrice: 50},
			{Description: "Oil needs to be changed", RepairPrice: 20},
			{Description: "Engine temperature sensor broken", RepairPrice: 90},
		}},
		{ID: "car4", Make: "Alfa Romeo", Model: "147", YearManufactured: 2004, Color: "red", OwnerID: "person1", TransferPrice: 2000, PaintPrice: 10, CarDefects: []CarDefect{
			{Description: "Engine broken", RepairPrice: 500},
			{Description: "Transmission broken", RepairPrice: 400},
		}},
		{ID: "car5", Make: "BMW", Model: "540i", YearManufactured: 2019, Color: "blue", OwnerID: "person1", TransferPrice: 5300, PaintPrice: 85, CarDefects: []CarDefect{
			{Description: "Oil needs to be changed", RepairPrice: 50},
		}},
		{ID: "car6", Make: "Mercedes", Model: "GLE", YearManufactured: 2017, Color: "white", OwnerID: "person2", TransferPrice: 6000, PaintPrice: 90, CarDefects: []CarDefect{}},
	}

	initPersons := []Person{
		{ID: "person1", Name: "John", Surname: "Johnson", Email: "john@john.com", MoneyLeft: 9000.0},
		{ID: "person2", Name: "Jane", Surname: "Doe", Email: "jane@jane.com", MoneyLeft: 5500.5},
		{ID: "person3", Name: "Rasmus", Surname: "Rasmussen", Email: "rasmus@rasmus.com", MoneyLeft: 1500.0},
	}

	for _, car := range initCars {
		carAsJSON, err := json.Marshal(car)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(car.ID, carAsJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state (cars). %v", err)
		}

		indexName := "color~owner~ID"
		colorOwnerIndexKey, err := ctx.GetStub().CreateCompositeKey(indexName, []string{car.Color, car.OwnerID, car.ID})
		if err != nil {
			return err
		}

		value := []byte{0x00}
		err = ctx.GetStub().PutState(colorOwnerIndexKey, value)
		if err != nil {
			return err
		}
	}

	for _, person := range initPersons {
		personAsJSON, err := json.Marshal(person)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(person.ID, personAsJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state (persons). %v", err)
		}
	}

	return nil
}

func (s *SmartContract) QueryCar(ctx contractapi.TransactionContextInterface, id string) (*Car, error) {
	carAsJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state (car): %v", err)
	}
	if carAsJSON == nil {
		return nil, fmt.Errorf("car %s does not exist", id)
	}

	var car Car
	err = json.Unmarshal(carAsJSON, &car)
	if err != nil {
		return nil, err
	}

	return &car, nil
}

func (s *SmartContract) QueryPerson(ctx contractapi.TransactionContextInterface, id string) (*Person, error) {
	personAsJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state (person): %v", err)
	}
	if personAsJSON == nil {
		return nil, fmt.Errorf("person %s does not exist", id)
	}

	var person Person
	err = json.Unmarshal(personAsJSON, &person)
	if err != nil {
		return nil, err
	}

	return &person, nil
}

func (s *SmartContract) TransferOwnershipOfCar(ctx contractapi.TransactionContextInterface, carID string, newOwnerID string, defectsAccepted bool) (bool, error) {
	car, err := s.QueryCar(ctx, carID)
	if err != nil {
		return false, err
	}

	newOwner, err := s.QueryPerson(ctx, newOwnerID)
	if err != nil {
		return false, err
	}

	oldOwner, err := s.QueryPerson(ctx, car.OwnerID)
	if err != nil {
		return false, err
	}

	if oldOwner.ID == newOwner.ID {
		return false, fmt.Errorf("Old and new owner are the same person!")
	}

	var totalTransferPrice float32

	if car.CarDefects == nil {
		totalTransferPrice = car.TransferPrice
	} else if len(car.CarDefects) == 0 {
		totalTransferPrice = car.TransferPrice
	} else if defectsAccepted {
		var totalRepairPrice float32

		for _, carDefect := range car.CarDefects {
			totalRepairPrice += carDefect.RepairPrice
		}

		totalTransferPrice = car.TransferPrice - totalRepairPrice
	} else {
		return false, fmt.Errorf("Car has defects!")
	}

	oldOwnerID := car.OwnerID

	car.OwnerID = newOwnerID

	if newOwner.MoneyLeft >= totalTransferPrice {
		newOwner.MoneyLeft -= totalTransferPrice
		oldOwner.MoneyLeft += totalTransferPrice
	} else {
		return false, fmt.Errorf("Specified new owner doesn't have enough money to buy this car!")
	}

	carAsJSON, err := json.Marshal(car)
	if err != nil {
		return false, err
	}

	newOwnerAsJSON, err := json.Marshal(newOwner)
	if err != nil {
		return false, err
	}

	oldOwnerAsJSON, err := json.Marshal(oldOwner)
	if err != nil {
		return false, err
	}

	err = ctx.GetStub().PutState(carID, carAsJSON)
	if err != nil {
		return false, err
	}

	err = ctx.GetStub().PutState(newOwner.ID, newOwnerAsJSON)
	if err != nil {
		return false, err
	}

	err = ctx.GetStub().PutState(oldOwner.ID, oldOwnerAsJSON)
	if err != nil {
		return false, err
	}

	indexName := "color~owner~ID"
	colorOwnerIndexKeyOld, err := ctx.GetStub().CreateCompositeKey(indexName, []string{car.Color, oldOwnerID, car.ID})
	if err != nil {
		return false, err
	}

	err = ctx.GetStub().DelState(colorOwnerIndexKeyOld)
	if err != nil {
		return false, err
	}

	colorOwnerIndexKeyNew, err := ctx.GetStub().CreateCompositeKey(indexName, []string{car.Color, newOwnerID, car.ID})
	if err != nil {
		return false, err
	}
	value := []byte{0x00}
	err = ctx.GetStub().PutState(colorOwnerIndexKeyNew, value)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *SmartContract) PaintCar(ctx contractapi.TransactionContextInterface, carID string, newColor string) error {
	car, err := s.QueryCar(ctx, carID)
	if err != nil {
		return err
	}

	owner, err := s.QueryPerson(ctx, car.OwnerID)
	if err != nil {
		return err
	}

	if owner.MoneyLeft < car.PaintPrice {
		return fmt.Errorf("Not enough money to paint carx!")
	}

	oldColor := car.Color
	car.Color = newColor

	carAsJSON, err := json.Marshal(car)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(carID, carAsJSON)
	if err != nil {
		return err
	}

	owner.MoneyLeft -= car.PaintPrice

	ownerAsJSON, err := json.Marshal(owner)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(owner.ID, ownerAsJSON)
	if err != nil {
		return err
	}

	indexName := "color~owner~ID"
	colorOwnerIndexKeyOld, err := ctx.GetStub().CreateCompositeKey(indexName, []string{oldColor, car.OwnerID, car.ID})
	if err != nil {
		return err
	}

	err = ctx.GetStub().DelState(colorOwnerIndexKeyOld)
	if err != nil {
		return err
	}

	colorOwnerIndexKeyNew, err := ctx.GetStub().CreateCompositeKey(indexName, []string{newColor, car.OwnerID, car.ID})
	if err != nil {
		return err
	}

	value := []byte{0x00}
	err = ctx.GetStub().PutState(colorOwnerIndexKeyNew, value)
	if err != nil {
		return err
	}

	return nil
}

func (s *SmartContract) CreateCarDefect(ctx contractapi.TransactionContextInterface, carID string, description string, repairPrice float32) error {
	car, err := s.QueryCar(ctx, carID)
	if err != nil {
		return err
	}

	newCarDefect := CarDefect{
		Description: description,
		RepairPrice: repairPrice,
	}

	car.CarDefects = append(car.CarDefects, newCarDefect)

	carAsJSON, err := json.Marshal(car)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(carID, carAsJSON)
	if err != nil {
		return err
	}

	var totalRepairPrice float32
	for _, carDefect := range car.CarDefects {
		totalRepairPrice += carDefect.RepairPrice
	}

	if totalRepairPrice > car.TransferPrice {
		ctx.GetStub().DelState(carID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SmartContract) RepairCar(ctx contractapi.TransactionContextInterface, id string) error {
	car, err := s.QueryCar(ctx, id)
	if err != nil {
		return err
	}

	owner, err := s.QueryPerson(ctx, car.OwnerID)
	if err != nil {
		return err
	}

	var totalRepairPrice float32
	for _, carDefect := range car.CarDefects {
		totalRepairPrice += carDefect.RepairPrice
	}

	if totalRepairPrice > owner.MoneyLeft {
		return fmt.Errorf("Owner doesn't have enough money to repair all car defects!")
	}
	owner.MoneyLeft -= totalRepairPrice

	car.CarDefects = []CarDefect{}

	carAsJSON, err := json.Marshal(car)
	if err != nil {
		return err
	}

	ownerAsJSON, err := json.Marshal(owner)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(id, carAsJSON)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(owner.ID, ownerAsJSON)
	if err != nil {
		return err
	}

	return nil
}

func (s *SmartContract) QueryCarsByColor(ctx contractapi.TransactionContextInterface, color string) ([]*Car, error) {
	carWithColorIter, err := ctx.GetStub().GetStateByPartialCompositeKey("color~owner~ID", []string{color})
	if err != nil {
		return nil, err
	}

	defer carWithColorIter.Close()

	result := make([]*Car, 0)

	for i := 0; carWithColorIter.HasNext(); i++ {
		responseRange, err := carWithColorIter.Next()
		if err != nil {
			return nil, err
		}

		_, compositeKeyParts, err := ctx.GetStub().SplitCompositeKey(responseRange.Key)
		if err != nil {
			return nil, err
		}

		resultCarID := compositeKeyParts[2]

		car, err := s.QueryCar(ctx, resultCarID)
		if err != nil {
			return nil, err
		}

		result = append(result, car)
	}

	return result, nil
}

func (s *SmartContract) QueryCarsByOwnerAndColor(ctx contractapi.TransactionContextInterface, ownerID string, color string) ([]*Car, error) {
	exists, err := s.PersonExists(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("Owner with that ID doesn't exist!")
	}

	carWithColorAndOwnerIter, err := ctx.GetStub().GetStateByPartialCompositeKey("color~owner~ID", []string{color, ownerID})
	if err != nil {
		return nil, err
	}

	defer carWithColorAndOwnerIter.Close()

	result := make([]*Car, 0)

	for i := 0; carWithColorAndOwnerIter.HasNext(); i++ {
		responseRange, err := carWithColorAndOwnerIter.Next()
		if err != nil {
			return nil, err
		}

		_, compositeKeyParts, err := ctx.GetStub().SplitCompositeKey(responseRange.Key)
		if err != nil {
			return nil, err
		}

		resultCarID := compositeKeyParts[2]

		car, err := s.QueryCar(ctx, resultCarID)
		if err != nil {
			return nil, err
		}

		result = append(result, car)
	}

	return result, nil
}

func (s *SmartContract) PersonExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	personAsJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state (person): %v", err)
	}

	return personAsJSON != nil, nil
}

func main() {
	assetChaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		log.Panicf("Error creating my chaincode: %v", err)
	}

	if err := assetChaincode.Start(); err != nil {
		log.Panicf("Error starting my chaincode: %v", err)
	}
}
