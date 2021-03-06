// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validation_test

import (
	apisalicloud "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud"
	. "github.com/gardener/gardener-extension-provider-alicloud/pkg/apis/alicloud/validation"
	"github.com/gardener/gardener/pkg/apis/core"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
)

var _ = Describe("Shoot validation", func() {
	Describe("#ValidateNetworking", func() {
		var networkingPath = field.NewPath("spec", "networking")

		It("should return no error because nodes CIDR was provided", func() {
			networking := core.Networking{
				Nodes: pointer.StringPtr("1.2.3.4/5"),
			}

			errorList := ValidateNetworking(networking, networkingPath)

			Expect(errorList).To(BeEmpty())
		})

		It("should return an error because no nodes CIDR was provided", func() {
			networking := core.Networking{}

			errorList := ValidateNetworking(networking, networkingPath)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.networking.nodes"),
				})),
			))
		})
	})

	Describe("#ValidateWorkerConfig", func() {
		var (
			workers       []core.Worker
			alicloudZones []apisalicloud.Zone
		)

		BeforeEach(func() {
			workers = []core.Worker{
				{
					Name: "worker1",
					Volume: &core.Volume{
						Type: pointer.StringPtr("Volume"),
						Size: "30G",
					},
					Zones: []string{
						"zone1",
						"zone2",
					},
				},
				{
					Name: "worker2",
					Volume: &core.Volume{
						Type: pointer.StringPtr("Volume"),
						Size: "20G",
					},
					Zones: []string{
						"zone2",
						"zone3",
					},
				},
			}

			alicloudZones = []apisalicloud.Zone{
				{
					Name:    "zone1",
					Workers: "1.2.3.4/5",
				},
				{
					Name:    "zone2",
					Workers: "1.2.3.4/5",
				},
				{
					Name:    "zone3",
					Workers: "1.2.3.4/5",
				},
			}
		})

		Describe("#ValidateWorkers", func() {
			It("should pass because workers are configured correctly", func() {
				errorList := ValidateWorkers(workers, alicloudZones, field.NewPath(""))

				Expect(errorList).To(BeEmpty())
			})

			It("should forbid because volume is not configured", func() {
				workers[0].Volume = nil

				errorList := ValidateWorkers(workers, alicloudZones, field.NewPath("workers"))

				Expect(errorList).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("workers[0].volume"),
					})),
				))
			})

			It("should forbid because volume type and size are not configured", func() {
				workers[0].Volume.Type = nil
				workers[0].Volume.Size = ""

				errorList := ValidateWorkers(workers, alicloudZones, field.NewPath("workers"))

				Expect(errorList).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("workers[0].volume.type"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("workers[0].volume.size"),
					})),
				))
			})

			It("should forbid because worker does not specify a zone", func() {
				workers[0].Zones = nil

				errorList := ValidateWorkers(workers, alicloudZones, field.NewPath("workers"))

				Expect(errorList).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal("workers[0].zones"),
					})),
				))
			})

			It("should forbid because worker use zones which are not available", func() {
				workers[0].Zones[0] = "zone4"
				workers[1].Zones[1] = "not-available"

				errorList := ValidateWorkers(workers, alicloudZones, field.NewPath("workers"))

				Expect(errorList).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("workers[0].zones[0]"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("workers[1].zones[1]"),
					})),
				))
			})
		})

		Describe("#ValidateWorkersUpdate", func() {
			It("should pass because workers are unchanged", func() {
				newWorkers := copyWorkers(workers)

				errorList := ValidateWorkersUpdate(workers, newWorkers, field.NewPath("workers"))

				Expect(errorList).To(BeEmpty())
			})

			It("should allow adding workers", func() {
				newWorkers := copyWorkers(workers)
				newWorkers = append(newWorkers, core.Worker{Name: "worker3", Zones: []string{"zone1"}})

				errorList := ValidateWorkersUpdate(workers, newWorkers, field.NewPath("workers"))

				Expect(errorList).To(BeEmpty())
			})

			It("should allow adding a zone to a worker", func() {
				newWorkers := copyWorkers(workers)
				newWorkers[0].Zones = append(newWorkers[0].Zones, "another-zone")

				errorList := ValidateWorkersUpdate(workers, newWorkers, field.NewPath("workers"))

				Expect(errorList).To(BeEmpty())
			})

			It("should forbid removing a zone from a worker", func() {
				newWorkers := copyWorkers(workers)
				newWorkers[1].Zones = newWorkers[1].Zones[1:]

				errorList := ValidateWorkersUpdate(workers, newWorkers, field.NewPath("workers"))

				Expect(errorList).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("workers[1].zones"),
					})),
				))
			})

			It("should forbid changing the zone order", func() {
				newWorkers := copyWorkers(workers)
				newWorkers[0].Zones[0] = workers[0].Zones[1]
				newWorkers[0].Zones[1] = workers[0].Zones[0]
				newWorkers[1].Zones[0] = workers[1].Zones[1]
				newWorkers[1].Zones[1] = workers[1].Zones[0]

				errorList := ValidateWorkersUpdate(workers, newWorkers, field.NewPath("workers"))

				Expect(errorList).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("workers[0].zones"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal("workers[1].zones"),
					})),
				))
			})
		})

		It("should forbid adding a zone while changing an existing one", func() {
			newWorkers := copyWorkers(workers)
			newWorkers = append(newWorkers, core.Worker{Name: "worker3", Zones: []string{"zone1"}})
			newWorkers[1].Zones[0] = workers[1].Zones[1]

			errorList := ValidateWorkersUpdate(workers, newWorkers, field.NewPath("workers"))

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("workers[1].zones"),
				})),
			))
		})
	})
})

func copyWorkers(workers []core.Worker) []core.Worker {
	copy := append(workers[:0:0], workers...)
	for i := range copy {
		copy[i].Zones = append(workers[i].Zones[:0:0], workers[i].Zones...)
	}
	return copy
}
