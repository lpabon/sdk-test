/*
Copyright 2018 Portworx

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sanity

import (
	//"google.golang.org/grpc/codes"
	//"google.golang.org/grpc/status"

	"context"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/libopenstorage/openstorage/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Enumerate [OpenStorageCluster]", func() {
	var (
		c api.OpenStorageClusterClient
		v api.OpenStorageVolumeClient

		volID            string
		numVolumesBefore int
		numVolumesAfter  int
	)

	BeforeEach(func() {
		c = api.NewOpenStorageClusterClient(conn)
		v = api.NewOpenStorageVolumeClient(conn)

		numVolumesBefore = numberOfVolumesInCluster(v)
		volID = ""
	})

	AfterEach(func() {
		if volID != "" {
			_, err := v.Delete(
				context.Background(),
				&api.SdkVolumeDeleteRequest{VolumeId: volID},
			)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("should return a cluster id", func() {
		info, err := c.InspectCurrent(context.Background(),
			&api.SdkClusterInspectCurrentRequest{})
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Cluster).NotTo(BeNil())
	})

	Describe("Enumerate Alerts[Openstorage]", func() {

		It("Should Enumerate Alerts for volume created / deleted", func() {
			By("Creating the volume first")

			var err error
			startTime := ptypes.TimestampNow()
			req := &api.SdkVolumeCreateRequest{
				Name: "sdk-vol",
				Spec: &api.VolumeSpec{
					Size:             uint64(5 * GIGABYTE),
					AggregationLevel: 2,
					Encrypted:        true,
					Shared:           false,
					HaLevel:          3,
					IoProfile:        api.IoProfile_IO_PROFILE_DB,
					Cos:              api.CosType_HIGH,
					Sticky:           true,
					Format:           api.FSType_FS_TYPE_XFS,
				},
			}

			vResp, err := v.Create(context.Background(), req)
			Expect(err).NotTo(HaveOccurred())

			// Test if no. of volumes increased by 1
			numVolumesAfter = numberOfVolumesInCluster(v)
			Expect(numVolumesAfter).To(BeEquivalentTo(numVolumesBefore + 1))

			// Test Volume Details by Calling Volume Inspect
			inspectReq := &api.SdkVolumeInspectRequest{
				VolumeId: vResp.VolumeId,
			}
			inspectResponse, err := v.Inspect(context.Background(), inspectReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(inspectResponse).NotTo(BeNil())

			// Test the details of the created volume
			testVolumeDetails(req, inspectResponse.Volume)
			volID = vResp.VolumeId

			By("Deleting the created volume")

			_, err = v.Delete(context.Background(),
				&api.SdkVolumeDeleteRequest{VolumeId: volID},
			)
			Expect(err).ToNot(HaveOccurred())
			volID = ""

			endTime := ptypes.TimestampNow()
			alerts, err := c.AlertEnumerate(context.Background(), &api.SdkClusterAlertEnumerateRequest{
				TimeStart: startTime,
				TimeEnd:   endTime,
				Resource:  api.ResourceType_RESOURCE_TYPE_VOLUME,
			})
			Expect(err).NotTo(HaveOccurred())

			noOfOccurence := 0
			for _, alert := range alerts.GetAlerts() {
				if alert.ResourceId == volID {
					noOfOccurence++
				}
			}
			// No of occurence should be 2  [one for create and one for delete]
			//  TBD: fake driver does not support alert for volumes operation
			//			Expect(noOfOccurence).To(BeEquivalentTo(2))
		})

		It("Should enumerate alerts for all resource types ", func() {

			By("Enumeraing alerts")

			endTime := ptypes.TimestampNow()
			startTime, _ := ptypes.TimestampProto(time.Now().Add(-5 * time.Hour))

			for _, v := range api.ResourceType_value {
				alerts, err := c.AlertEnumerate(context.Background(), &api.SdkClusterAlertEnumerateRequest{
					TimeStart: startTime,
					TimeEnd:   endTime,
					Resource:  api.ResourceType(v),
				})

				//startTime, endTime, api.ResourceType(v))
				Expect(err).NotTo(HaveOccurred())
				Expect(alerts).NotTo(BeNil())
			}
		})
	})

	Describe("Clear and Erase Alerts", func() {

		It("Should clear and erase alerts", func() {

			By("Taking a random alertID from volume resource type")

			endTime := ptypes.TimestampNow()
			startTime, _ := ptypes.TimestampProto(time.Now().Add(-5 * time.Hour))

			alerts, err := c.AlertEnumerate(context.Background(), &api.SdkClusterAlertEnumerateRequest{
				TimeStart: startTime,
				TimeEnd:   endTime,
				Resource:  api.ResourceType_RESOURCE_TYPE_NODE,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(alerts).NotTo(BeNil())
			//TBD: add support in fake driver for alert
			//			randomVolumeAlertID := alerts.GetAlerts()[random(0, len(alerts.GetAlerts()))].Id
			/*
				By("Clear alerts")
				_, err = c.AlertClear(
					context.Background(),
					&api.SdkClusterAlertClearRequest{
						AlertId:  randomVolumeAlertID,
						Resource: api.ResourceType_RESOURCE_TYPE_NODE,
					},
				)
				Expect(err).NotTo(HaveOccurred())

				By("Enumerating the alerts again and checking if the alert cleared")

				alerts, err = c.AlertEnumerate(context.Background(), &api.SdkClusterAlertEnumerateRequest{
					TimeStart: startTime,
					TimeEnd:   endTime,
					Resource:  api.ResourceType_RESOURCE_TYPE_NODE,
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(alerts).NotTo(BeNil())

				for _, alert := range alerts.GetAlerts() {
					if alert.Id == randomVolumeAlertID {
						Expect(alert.Cleared).To(BeTrue())
						break
					}
				}

				By("Erasing alerts")
				_, err = c.AlertDelete(context.Background(),
					&api.SdkClusterAlertDeleteRequest{
						AlertId:  randomVolumeAlertID,
						Resource: api.ResourceType_RESOURCE_TYPE_NODE,
					},
				)
				Expect(err).NotTo(HaveOccurred())

				By("Enumerating the alerts again and checking if the alert cleared")

				alerts, err = c.AlertEnumerate(context.Background(), &api.SdkClusterAlertEnumerateRequest{
					TimeStart: startTime,
					TimeEnd:   endTime,
					Resource:  api.ResourceType_RESOURCE_TYPE_NODE,
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(alerts).NotTo(BeNil())

							noOfOccurence := 0
							for _, alert := range alerts.GetAlerts() {
								if alert.Id == randomVolumeAlertID {
									noOfOccurence++
								}
							}
				// Alert should not present
				//Expect(noOfOccurence).To(BeEquivalentTo(0))
			*/
		})

	})
})
