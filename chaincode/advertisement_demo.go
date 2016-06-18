package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"strconv"
)

//==============================================================================================================================
//	 Structure Definitions
//==============================================================================================================================
//	SimpleChaincode - A blank struct for use with Shim (An IBM Blockchain included go file used for get/put state
//					  and other IBM Blockchain functions)
//==============================================================================================================================
type SimpleChaincode struct {
}

type Slot struct {
	Id						string			`json:"id"`
	StartTime     			int64 			`json:"startTime"`
	EndTime		 			int64 			`json:"endTime"`
	Bids					[]Bid 			`json:"bids"`
	BidDeadline				int64			`json:"bidDeadline"`
	DeviceId				string			`json:"deviceId"`
	HighestBidAmount		int64			`json:"highestBidAmount"`
}

type Bid struct {
	Id						string			`json:"id"`
	SlotId					string			`json:"slotId"`
	Amount					int64			`json:"amount"`
	Content					string			`json:"content"`
	Username				string			`json:"username"`
}

type Account struct {
	Hash					string			`json:"hash"`
	Salt					string			`json:"salt"`
	Username				string			`json:"username"`
	Balance					int64			`json:"balance"`
	Bids					[]Bid			`json:"bids"`
}

type Device struct {
	DeviceId				string			`json:"deviceId"`
	Hash					string			`json:"hash"`
	Salt					string			`json:"salt"`
	Balance					string			`json:"balance"`
	Size					string			`json:"size"`
	Lat						string			`json:"lat"`
	Long					string			`json:"long"`
}

type Payment struct {
	RecipientId				string			`json:"recipient"`
	SenderId				string			`json:"sender"`
	Amount					int64			`json:"amount"`
}

//=================================================================================================================================
//  Index collections - In order to create new IDs dynamically and in progressive sorting
//=================================================================================================================================
var accountIndexStr = "_accounts"
var slotIndexStr = "_slots"
var bidIndexStr = "_bids"
var deviceIndexStr = "_devices"
//var paymentIndexStr = "_payments"

//==============================================================================================================================
//	Run - Called on chaincode invoke. Takes a function name passed and calls that function. Converts some
//		  initial arguments passed to other things for use in the called function e.g. name -> ecert
//==============================================================================================================================
func (t *SimpleChaincode) Run(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("run is running " + function)
	return t.Invoke(stub, function, args)
}

func (t *SimpleChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	if function == "init" {
		return t.Init(stub, "init", args)
	} else if function == "add_slot" {
		return t.add_slot(stub, args)
	} else if function == "place_bid" {
		return t.place_bid(stub, args)
	} else if function == "payout_bid" {
		return t.payout_bid(stub, args)
	} else if function == "add_device" {
		return t.add_device(stub, args)
	} else if function == "add_account" {
		return t.add_account(stub, args)
	}

	return nil, errors.New("Received unknown invoke function name")
}

//=================================================================================================================================
//	Query - Called on chaincode query. Takes a function name passed and calls that function. Passes the
//  		initial arguments passed are passed on to the called function.
//=================================================================================================================================
func (t *SimpleChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {

	if function == "get_account" {
		return t.get_account(stub, args[1])
	} else if function == "get_all_slots" {
		return t.get_all_slots(stub, args)
	} else if function == "get_all_bids" {
		return t.get_all_bids(stub, args)
	} else if function == "get_slot" {
		return t.get_slot(stub, args)
	} else if function == "get_device" {
		return t.get_device(stub, args)
	}

	return nil, errors.New("Received unknown query function name")
}

//==============================================================================================================================
//  Invoke Functions
//==============================================================================================================================

func (t *SimpleChaincode) add_account(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	//Args
	//			0				1
	//		  username		account JSON object (as string)

	id, err := append_id(stub, accountIndexStr, args[0], false)
	if err != nil {
		return nil, errors.New("Error creating new id for user " + args[0])
	}

	err = stub.PutState(string(id), []byte(args[1]))
	if err != nil {
		return nil, errors.New("Error putting user data on ledger")
	}

	return nil, nil
}

func (t *SimpleChaincode) add_slot(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	//Args
	//			0				1
	//		  index		Slot JSON object (as string)
	id, err:= append_id(stub, slotIndexStr, args[0], false)
	if err != nil {
		return nil, errors.New("Error creating new slot with id " + args[0])
	}

	err = stub.PutState(string(id), []byte(args[1]))
	if err != nil {
		return nil, errors.New("Error putting slot on ledger")
	}

	return nil, nil

}

func (t *SimpleChaincode) add_device(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	//Args
	//			0				1
	//		  index		device JSON object (as string)

	id, err:= append_id(stub, deviceIndexStr, args[0], false)
	if err != nil {
		return nil, errors.New("Error creating new device with id " + args[0])
	}

	err = stub.PutState(string(id), []byte(args[1]))
	if err != nil {
		return nil, errors.New("Error putting device on ledger")
	}

	return nil, nil

}

func (t *SimpleChaincode) place_bid(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	//Args
	//			0				1
	//		  index			bid as JSON


	// 0. Check if user can place bid
	// 0a. Unmarshal bid JSON
	var b Bid
	err := json.Unmarshal([]byte(args[1]), &b)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Failed to Unmarshal payload of ID: " + args[0])
	}
	// 0b: fetch account
	accountBytes, err := stub.GetState(b.Username)

	if err != nil { return nil,errors.New("Could not retrieve information for this account")}

	var a Account
	err = json.Unmarshal(accountBytes, &a)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Failed to unmarshal account of " + b.Username)
	}

	// 0c. Get slot
	slotBytes, err := stub.GetState(b.SlotId)
	if err != nil { return nil, errors.New("Could not retrieve slot")	}
	// 0d. Unmarshal slot
	var s Slot
	err = json.Unmarshal(slotBytes, &s)
	if err != nil { return nil, errors.New("Error unmarshalling slot with ID: " + b.SlotId)}

	// 0e. Check: User has enough balance
	if a.Balance < b.Amount {
		return nil, errors.New("Unsufficient balance to place bid.")
	}

	// 0f. Check: If bid is equal or lower than highest bid amount, dismiss
	if s.HighestBidAmount >= b.Amount {
		return nil, errors.New("Bid is not high enough")
	} else {
		s.HighestBidAmount = b.Amount
	}

	// 0e. Take bid amount out of balance of account
	a.Balance = a.Balance - b.Amount

	// 1. append new bid
	id, err:= append_id(stub, bidIndexStr, b.Id, false)
	if err != nil { return nil, errors.New("Error creating new slot with id " + args[0]) }

	// 2. put new bid to state
	err = stub.PutState(string(id), []byte(args[1]))
	if err != nil { return nil, errors.New("Error putting slot on ledger") }

	// 3c. Append bid to bids on slot and account
	s.Bids = append(s.Bids, b)
	a.Bids = append(a.Bids, b)

	//Convert slot object to JSON
	slotAsBytes, err := json.Marshal(s)
	if err != nil { return nil, errors.New("Error marshalling slot")}

	// Storing the updated slot
	err = stub.PutState(s.Id, []byte(slotAsBytes))
	if err!= nil { return nil, errors.New("Error putting slot back on ledger after adding bid") }

	// Place Account back on state
	accountAsBytes, err := json.Marshal(a)
	if err != nil { return nil, errors.New("Error marshalling account")}

	// Storing the updated slot
	err = stub.PutState(a.Username, []byte(accountAsBytes))
	if err!= nil { return nil, errors.New("Error putting Account back on ledger after placing bid") }

	return nil, nil
}

func (t *SimpleChaincode) payout_bid(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	// Args
	// 		1
	//	   slotId

	// 1. Get slot
	// 1a. fetch slot
	slotAsBytes, err := stub.GetState(args[0])
	if err != nil { return nil, errors.New("Could not retrieve slot") }

	// 1b. unmarshal slot
	var s Slot
	err = json.Unmarshal(slotAsBytes, &s)
	if err != nil { return nil, errors.New("Error Unmarshalling slot with ID: " + s.Id )}

	// 2. loop through bids
	for _, bid := range s.Bids {
		// 3. return amounts to non-winning bid
		if bid.Amount < s.HighestBidAmount {
			// 3a. get account
			accountAsBytes, err := stub.GetState(bid.Username)
			if err != nil { return nil, errors.New("Could not get account while processing bid ")}

			var a Account
			err = json.Unmarshal(accountAsBytes, &a)
			if err != nil { return nil, errors.New("Error unmarshalling account with username: " + bid.Username)}

			// 3b. add amount back on account balance
			a.Balance += bid.Amount

			// 3c. Marshall & Place account back on ledger
			// Place Account back on state
			accountAsBytes, err = json.Marshal(a)
			if err != nil { return nil, errors.New("Error marshalling account")}

			// Storing the updated slot
			err = stub.PutState(a.Username, []byte(accountAsBytes))
			if err!= nil { return nil, errors.New("Error putting Account back on ledger after paying back balance") }
		}
	}

	return nil, nil
}

//==============================================================================================================================
//		Query Functions
//==============================================================================================================================

func (t *SimpleChaincode) get_account(stub *shim.ChaincodeStub, username string) ([]byte, error) {

	bytes, err := stub.GetState(username)

	if err != nil {
		return nil, errors.New("Could not retrieve information for this user")
	}

	return bytes, nil

}

func (t *SimpleChaincode) get_all_slots(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	slotsIndexBytes, err := stub.GetState(slotIndexStr)
	if err != nil { return nil, errors.New("Failed to get slots index")}

	var slotIndex []string
	err = json.Unmarshal(slotsIndexBytes, &slotIndex)
	if err != nil { return nil, errors.New("Could not marshal slot indexes") }

	var slots []Slot
	for _, slotId := range slotIndex {
		bytes, err := stub.GetState(slotId)
		if err != nil { return nil, errors.New("Not able to get slot") }

		var s Slot
		err = json.Unmarshal(bytes, &s)
		slots = append(slots, s)
	}

	slotsJson, err := json.Marshal(slots)
	if err != nil { return nil, errors.New("Failed to marshal slots to JSON")}

	return slotsJson, nil

}

func (t *SimpleChaincode) get_all_bids(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	bidIndexBytes, err := stub.GetState(bidIndexStr)
	if err != nil { return nil, errors.New("Failed to get bids index")}

	var bidIndex []string
	err = json.Unmarshal(bidIndexBytes, &bidIndex)
	if err != nil { return nil, errors.New("Could not marshal bid indexes") }

	var bids []Bid
	for _, bidId := range bidIndex {
		bytes, err := stub.GetState(bidId)
		if err != nil { return nil, errors.New("Not able to get bid") }

		var b Bid
		err = json.Unmarshal(bytes, &b)
		bids = append(bids, b)
	}

	bidsJson, err := json.Marshal(bids)
	if err != nil { return nil, errors.New("Failed to marshal bids to JSON")}

	return bidsJson, nil

}

func (t *SimpleChaincode) get_slot(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	//Args
	//			0
	//		slotID

	bytes, err := stub.GetState(args[0])

	if err != nil { return nil, errors.New("Error getting slot from ledger") }

	return bytes, nil

}

func (t *SimpleChaincode) get_device(stub *shim.ChaincodeStub, args []string) ([]byte, error) {

	//Args
	//			0
	//		deviceID

	bytes, err := stub.GetState(args[0])

	if err != nil { return nil, errors.New("Error getting device from ledger") }

	return bytes, nil

}


//=================================================================================================================================
//  Main - main - Starts up the chaincode
//=================================================================================================================================

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting SimpleChaincode: %s", err)
	}
}

//==============================================================================================================================
//  Init Function - Called when the user deploys the chaincode
//==============================================================================================================================

func (t *SimpleChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	return nil, nil
}

//==============================================================================================================================
//  Utility Functions
//==============================================================================================================================

// "create":  true -> create new ID, false -> append the id
func append_id(stub *shim.ChaincodeStub, indexStr string, id string, create bool) ([]byte, error) {

	indexAsBytes, err := stub.GetState(indexStr)
	if err != nil {
		return nil, errors.New("Failed to get " + indexStr)
	}
	fmt.Println(indexStr + " retrieved")

	// Unmarshal the index
	var tmpIndex []string
	json.Unmarshal(indexAsBytes, &tmpIndex)
	fmt.Println(indexStr + " unmarshalled")

	// Create new id
	var newId = id
	if create {
		newId += strconv.Itoa(len(tmpIndex) + 1)
	}

	// append the new id to the index
	tmpIndex = append(tmpIndex, newId)
	jsonAsBytes, _ := json.Marshal(tmpIndex)
	err = stub.PutState(indexStr, jsonAsBytes)
	if err != nil {
		return nil, errors.New("Error storing new " + indexStr + " into ledger")
	}

	return []byte(newId), nil

}
