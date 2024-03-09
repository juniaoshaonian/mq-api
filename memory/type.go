// Copyright 2021 ecodeclub
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

package memory

// PartitionIDGetter 此抽象用于Producer获取对应分区号
type PartitionIDGetter interface {
	// GetPartitionId 用于Producer获取分区号,返回值就是分区号
	GetPartitionID(key string) int64
}

// ConsumerPartitionAssigner 此抽象是给消费组使用，用于将分区分配给消费组内的消费者。
type ConsumerPartitionAssigner interface {
	// AssignPartition consumerList为消费组内的所有消费者，
	// partitions表示分区数，返回值为map[name][]int name对应consumerList的索引，对应的值消费者可消费的分区
	AssignPartition(consumers []string, partitions int) map[string][]int
}
