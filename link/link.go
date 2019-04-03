// Copyright 2018 Andrew Bates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package link

import (
	"errors"
	"time"

	"github.com/abates/insteon"
)

var (
	// ErrAlreadyLinked is returned when creating a link and an existing matching link is found
	ErrAlreadyLinked = errors.New("Responder already linked to controller")
)

// FindDuplicateLinks will perform a linear search of the
// LinkDB and return any links that are duplicates. Duplicate
// links are those that are equivalent as reported by LinkRecord.Equal
func FindDuplicateLinks(linkable insteon.Linkable) ([]*insteon.LinkRecord, error) {
	duplicates := make([]*insteon.LinkRecord, 0)
	links, err := linkable.Links()
	if err == nil {
		for i, l1 := range links {
			for _, l2 := range links[i+1:] {
				if l1.Equal(l2) {
					duplicates = append(duplicates, l2)
				}
			}
		}
	}
	return duplicates, err
}

// FindLinkRecord will perform a linear search of the database and return
// a LinkRecord that matches the group, address and controller/responder
// indicator
func FindLinkRecord(linkable insteon.Linkable, controller bool, address insteon.Address, group insteon.Group) (*insteon.LinkRecord, error) {
	links, err := linkable.Links()
	if err == nil {
		for _, link := range links {
			if link.Flags.Controller() == controller && link.Address == address && link.Group == group {
				return link, nil
			}
		}
	}
	return nil, err
}

// CrossLinkAll will create bi-directional links among all the devices
// listed. This is useful for creating virtual N-Way connections
func CrossLinkAll(group insteon.Group, linkable ...insteon.AddressableLinkable) error {
	for i, l1 := range linkable {
		for _, l2 := range linkable[i:] {
			if l1 != l2 {
				err := CrossLink(group, l1, l2)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// CrossLink will create bi-directional links between the two linkable
// devices. Each device will get both a controller and responder
// link for the given group. When using lighting control devices, this
// will effectively create a 3-Way light switch configuration
func CrossLink(group insteon.Group, l1, l2 insteon.AddressableLinkable) error {
	err := Link(group, l1, l2)
	if err == nil || err == ErrAlreadyLinked {
		err = Link(group, l2, l1)
		if err == ErrAlreadyLinked {
			err = nil
		}
	}

	return err
}

// ForceLink will create links in the controller and responder All-Link
// databases without first checking if the links exist. The links are
// created by simulating set button presses (using EnterLinkingMode)
func ForceLink(group insteon.Group, controller, responder insteon.Linkable) (err error) {
	insteon.Log.Debugf("Putting controller %s into linking mode", controller)
	// controller enters all-linking mode
	err = controller.EnterLinkingMode(group)
	defer controller.ExitLinkingMode()
	time.Sleep(2 * time.Second)

	// responder pushes the set button responder
	if err == nil {
		insteon.Log.Debugf("Assigning responder to group")
		err = responder.EnterLinkingMode(group)
		defer responder.ExitLinkingMode()
	}
	time.Sleep(time.Second)
	return
}

// UnlinkAll will unlink all groups between a controller and
// a responder device
func UnlinkAll(controller, responder insteon.AddressableLinkable) (err error) {
	links, err := controller.Links()
	if err == nil {
		for _, link := range links {
			if link.Address == responder.Address() {
				err = Unlink(link.Group, responder, controller)
			}
		}
	}
	return err
}

// Unlink will unlink a controller from a responder for a given Group. The
// controller is put into UnlinkingMode (analogous to unlinking mode via
// the set button) and then the responder is put into unlinking mode (also
// analogous to the set button pressed)
func Unlink(group insteon.Group, controller, responder insteon.Linkable) (err error) {
	// controller enters all-linking mode
	err = controller.EnterUnlinkingMode(group)
	defer controller.ExitLinkingMode()

	// wait a moment for messages to propagate
	time.Sleep(2 * time.Second)

	// responder pushes the set button responder
	if err == nil {
		insteon.Log.Debugf("Unlinking responder from group")
		err = responder.EnterLinkingMode(group)
		defer responder.ExitLinkingMode()
	}

	// wait a moment for messages to propagate
	time.Sleep(time.Second)

	return
}

// Link will add appropriate entries to the controller's and responder's All-Link
// database. Each devices' ALDB will be searched for existing links, if both entries
// exist (a controller link and a responder link) then nothing is done. If only one
// entry exists than the other is deleted and new links are created. Once the link
// check/cleanup has taken place the new links are created using ForceLink
func Link(group insteon.Group, controller, responder insteon.AddressableLinkable) (err error) {
	insteon.Log.Debugf("Looking for existing links")
	var controllerLink *insteon.LinkRecord
	controllerLink, err = FindLinkRecord(controller, true, responder.Address(), group)

	if err == nil {
		var responderLink *insteon.LinkRecord
		responderLink, err = FindLinkRecord(responder, false, controller.Address(), group)

		if err == nil {
			if controllerLink != nil && responderLink != nil {
				err = ErrAlreadyLinked
			} else {
				// correct a mismatch by deleting the one link found
				// and recreating both
				if controllerLink != nil {
					insteon.Log.Debugf("Controller link already exists, deleting it")
					err = controller.RemoveLinks(controllerLink)
				}

				if err == nil && responderLink != nil {
					insteon.Log.Debugf("Responder link already exists, deleting it")
					err = responder.RemoveLinks(controllerLink)
				}

				ForceLink(group, controller, responder)
			}
		}
	}
	return err
}
