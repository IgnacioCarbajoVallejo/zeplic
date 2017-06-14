// Package director contains: agent.go - !director.go - slave.go
//
// Slave receives a snapshot from agent
//
package director

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/nfrance-conseil/zeplic/lib"
	"github.com/mistifyio/go-zfs"
)

// Struct for ZFS orders from agent
// Case: send_snapshot
type ZFSOrderFromAgent struct {
	Source		string // hostname of agent
	OrderUUID	string // mandatory
	SnapshotUUID	string // uuid of snapshot received
	SnapshotName	string // name of snapshot received
	DestDataset	string // dataset for receive
}

// Struc for ZFS response to agent
type ZFSResponseToAgent struct {
	OrderUUID	string	`json:"OrderUUID"`
	IsSuccess	bool	`json:"IsSuccess"`
	Status		int64	`json:"Status"`
	Error		string	`json:"Error"`
}

// Handle incoming requests from agent
func HandleRequestSlave (connSlave net.Conn) bool {
	// Resolve hostname
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Printf("[ERROR] it was not possible to resolve the hostname.\n")
	}

	// Unmarshal orders from agent
	var a ZFSOrderFromAgent
	agent, err := bufio.NewReader(connSlave).ReadBytes('\x0A')
	if err != nil {
		fmt.Printf("[ERROR] an error has occurred while reading from the socket.\n")
	}
	err = json.Unmarshal(agent, &a)
	if err != nil {
		fmt.Printf("[ERROR] it was impossible to parse the JSON struct from the socket.\n")
	}

	// Struct for Status constant
	ack := make([]byte, 0)
	// Variable to receive an incremental stream
	stream := false

	// Check if the dataset received exists
	_, err = zfs.GetDataset(a.DestDataset)
	// Get the last snapshot in DestDataset
	list, _ := zfs.Snapshots(a.DestDataset)
	count := len(list)

	// Struct for response
	ResponseToAgent := ZFSResponseToAgent{}
	// Dataset does not exist
	if err != nil {
		// Status for DestDataset
		ack = nil
		ack = strconv.AppendInt(ack, DATASET_FALSE, 10)
		connSlave.Write(ack)

		// Receive the snapshot
		_, err := zfs.ReceiveSnapshotRollback(connSlave, a.DestDataset, false)

		// Check for response to agent
		if err != nil {
			Error := fmt.Sprintf("[ERROR from '%s'] it was not possible to receive the snapshot '%s' from '%s'.", hostname, a.SnapshotName, a.Source)
			ResponseToAgent = ZFSResponseToAgent{a.OrderUUID,false,ZFS_ERROR,Error}
			fmt.Printf("[ERROR] it was not possible to receive the snapshot '%s' from '%s'.\n", a.SnapshotName, a.Source)
		} else {
			ResponseToAgent = ZFSResponseToAgent{a.OrderUUID,true,WAS_WRITTEN,""}
			fmt.Printf("[INFO] the snapshot '%s' has been received.\n", a.SnapshotName)
		}

	} else {
		// Dataset is empty
		if count == 0 {
			// Status for DestDataset
			ack = nil
			ack = strconv.AppendInt(ack, DATASET_FALSE, 10)
			connSlave.Write(ack)

			// Receive the snapshot
			_, err := zfs.ReceiveSnapshotRollback(connSlave, a.DestDataset, true)

			// Check for response to agent
			if err != nil {
				Error := fmt.Sprintf("[ERROR from '%s'] it was not possible to receive the snapshot '%s' from '%s'.", hostname, a.SnapshotName, a.Source)
				ResponseToAgent = ZFSResponseToAgent{a.OrderUUID,false,ZFS_ERROR,Error}
				fmt.Printf("[ERROR] it was not possible to receive the snapshot '%s' from '%s'.\n", a.SnapshotName, a.Source)
			} else {
				ResponseToAgent = ZFSResponseToAgent{a.OrderUUID,true,WAS_WRITTEN,""}
				fmt.Printf("[INFO] the snapshot '%s' has been received.\n", a.SnapshotName)
			}
		} else {
			// Status for DestDataset
			ack = nil
			ack = strconv.AppendInt(ack, DATASET_TRUE, 10)
			connSlave.Write(ack)

			// Get the last snapshot in DestDataset
			LastSnapshotName := list[count-1].Name
			// Get its uuid
			LastSnapshotUUID := lib.SearchUUID(LastSnapshotName)

			// Check if the snapshot was renamed
			renamed := lib.Renamed(a.SnapshotName, LastSnapshotName)
			if LastSnapshotUUID == a.SnapshotUUID {
				if renamed == true {
					ResponseToAgent = ZFSResponseToAgent{a.OrderUUID,true,WAS_RENAMED,""}
					fmt.Printf("[INFO] the snapshot '%s' already existed but it was renamed to '%s'.\n", a.SnapshotName, LastSnapshotName)
				} else {
					ResponseToAgent = ZFSResponseToAgent{a.OrderUUID,true,NOTHING_TO_DO,""}
					fmt.Printf("[INFO] the snapshot '%s' already existed.\n", LastSnapshotName)
				}
			} else {
				// Information to agent where Error field contains the uuid of last snapshot in slave
				ResponseToAgent = ZFSResponseToAgent{a.OrderUUID,false,NOT_EMPTY,LastSnapshotUUID}
				stream = true
			}
		}
	}

	// Reconnection to send ZFSResponseToAgent
	connToAgent, err := net.Dial("tcp", a.Source+":7733")

	// Marshal response to agent
	rta, err := json.Marshal(ResponseToAgent)
	if err != nil {
		fmt.Printf("[ERROR] it was impossible to enconde the JSON struct.\n")
	} else {
		connToAgent.Write([]byte(rta))
		connToAgent.Write([]byte("\n"))
		connToAgent.Close()
	}

	if stream == true {
		l2, _ := net.Listen("tcp", ":7744")
		defer l2.Close()
		fmt.Println("[SLAVE:7744] Receiving incremental stream from agent...")

		conn2Slave, _ := l2.Accept()

		// Read the status
		buff := bufio.NewReader(conn2Slave)
		n, _ := buff.ReadByte()
		snapExist, _ := strconv.Atoi(string(n))

		// Last snapshot in slave node
		LastSnapshotName := list[count-1].Name

		switch snapExist {
		// Case: the most actual snapshot in slave is not correlative
		case ZFS_ERROR:
			Error := fmt.Sprintf("[ERROR from '%s'] the most actual snapshot '%s' is not correlative with the snapshot sent.", hostname, LastSnapshotName)
			ResponseToAgent = ZFSResponseToAgent{a.OrderUUID,false,ZFS_ERROR,Error}
			fmt.Printf("[ERROR] the snapshot '%s' is not correlative.\n", LastSnapshotName)

		// Case: the last snapshot in slave is the most actual
		case MOST_ACTUAL:
			ResponseToAgent = ZFSResponseToAgent{a.OrderUUID,true,NOTHING_TO_DO,""}
			fmt.Printf("[INFO] the snapshot '%s' is the most actual.\n", LastSnapshotName)

		// Case: receive incremental stream
		case INCREMENTAL:
			// Receive incremental stream
			zfs.ReceiveSnapshotRollback(conn2Slave,a.DestDataset,true)

			// Check for response to agent
			if err != nil {
				Error := fmt.Sprintf("[ERROR from '%s'] it was not possible to receive the snapshot '%s' from '%s'.", hostname, a.SnapshotName, a.Source)
				ResponseToAgent = ZFSResponseToAgent{a.OrderUUID,false,ZFS_ERROR,Error}
				fmt.Printf("[ERROR] it was not possible to receive the snapshot '%s' from '%s'.\n", a.SnapshotName, a.Source)
			} else {
				ResponseToAgent = ZFSResponseToAgent{a.OrderUUID,true,WAS_WRITTEN,""}
				fmt.Printf("[INFO] the snapshot '%s' has been received.\n", a.SnapshotName)
			}
		}
		// Send the last ZFSResponseToAgent
		conn2ToAgent, err := net.Dial("tcp", a.Source+":7755")

		// Marshal response to agent
		rta, err = json.Marshal(ResponseToAgent)
		if err != nil {
			fmt.Printf("[ERROR] it was impossible to encode the JSON struct.\n")
		} else {
			conn2ToAgent.Write([]byte(rta))
			conn2ToAgent.Write([]byte("\n"))
			conn2ToAgent.Close()
		}
		// Close transmission
		stream = false
	}
	stop := false
	return stop
}
