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

//go:build e2e

package e2e

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/mq-api/mqerr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/multierr"
)

type MqCreator interface {
	Init() mq.MQ
	Ping(ctx context.Context) error
}

type TestSuite struct {
	suite.Suite
	testMqCreator MqCreator
	testMq        mq.MQ
}

type ProducerMsg struct {
	partition int32
	msg       *mq.Message
}

func NewBaseSuite(mq MqCreator, testMq mq.MQ) *TestSuite {
	return &TestSuite{
		testMqCreator: mq,
		testMq:        testMq,
	}
}

func (b *TestSuite) SetupSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	if err := b.testMqCreator.Ping(ctx); err != nil {
		panic("第三方依赖连接不上")
	}
	cancel()
	time.Sleep(1 * time.Second)
}

// 测试消费组
func (b *TestSuite) TestMQConsumer_ConsumerGroup() {
	testcases := []struct {
		name        string
		topic       string
		partitions  int64
		input       []*mq.Message
		consumers   func(mqm mq.MQ) []mq.Consumer
		consumeFunc func(c mq.Consumer) []*mq.Message
		wantVal     []*mq.Message
	}{
		{
			name:       "多个消费组订阅同一个Topic,消费组之间可以重复消费消息",
			topic:      "test_topic1",
			partitions: 4,
			input: []*mq.Message{
				{
					Value: []byte("1"),
					Key:   []byte("1"),
				},
				{
					Value: []byte("2"),
					Key:   []byte("2"),
				},
				{
					Value: []byte("3"),
					Key:   []byte("3"),
				},
				{
					Value: []byte("4"),
					Key:   []byte("4"),
				},
				{
					Value: []byte("5"),
					Key:   []byte("5"),
				},
			},
			consumers: func(mqm mq.MQ) []mq.Consumer {
				c11, err := mqm.Consumer("test_topic1", "c1")
				require.NoError(b.T(), err)
				c12, err := mqm.Consumer("test_topic1", "c1")
				require.NoError(b.T(), err)
				c13, err := mqm.Consumer("test_topic1", "c1")
				require.NoError(b.T(), err)
				c21, err := mqm.Consumer("test_topic1", "c2")
				require.NoError(b.T(), err)
				c22, err := mqm.Consumer("test_topic1", "c2")
				require.NoError(b.T(), err)
				c23, err := mqm.Consumer("test_topic1", "c2")
				require.NoError(b.T(), err)
				return []mq.Consumer{
					c11,
					c12,
					c13,
					c21,
					c22,
					c23,
				}
			},
			consumeFunc: func(c mq.Consumer) []*mq.Message {
				msgCh, err := c.ConsumeChan(context.Background())
				require.NoError(b.T(), err)
				msgs := make([]*mq.Message, 0, 32)
				for val := range msgCh {
					msgs = append(msgs, val)
				}
				return msgs
			},
			wantVal: []*mq.Message{
				{
					Value: []byte("1"),
					Key:   []byte("1"),
					Topic: "test_topic1",
				},
				{
					Value: []byte("2"),
					Key:   []byte("2"),
					Topic: "test_topic1",
				},
				{
					Value: []byte("3"),
					Key:   []byte("3"),
					Topic: "test_topic1",
				},
				{
					Value: []byte("4"),
					Key:   []byte("4"),
					Topic: "test_topic1",
				},
				{
					Value: []byte("5"),
					Key:   []byte("5"),
					Topic: "test_topic1",
				},
				{
					Value: []byte("1"),
					Key:   []byte("1"),
					Topic: "test_topic1",
				},
				{
					Value: []byte("2"),
					Key:   []byte("2"),
					Topic: "test_topic1",
				},
				{
					Value: []byte("3"),
					Key:   []byte("3"),
					Topic: "test_topic1",
				},
				{
					Value: []byte("4"),
					Key:   []byte("4"),
					Topic: "test_topic1",
				},
				{
					Value: []byte("5"),
					Key:   []byte("5"),
					Topic: "test_topic1",
				},
			},
		},
		{
			name:       "同一消费者组内,各个消费者竞争消费消息",
			topic:      "test_topic2",
			partitions: 4,
			input: []*mq.Message{
				{
					Value: []byte("1"),
					Key:   []byte("1"),
				},
				{
					Value: []byte("2"),
					Key:   []byte("2"),
				},
				{
					Value: []byte("3"),
					Key:   []byte("3"),
				},
				{
					Value: []byte("4"),
					Key:   []byte("4"),
				},
				{
					Value: []byte("5"),
					Key:   []byte("5"),
				},
			},
			consumers: func(mqm mq.MQ) []mq.Consumer {
				c11, err := mqm.Consumer("test_topic2", "c1")
				require.NoError(b.T(), err)
				c12, err := mqm.Consumer("test_topic2", "c1")
				require.NoError(b.T(), err)
				c13, err := mqm.Consumer("test_topic2", "c1")
				require.NoError(b.T(), err)
				c14, err := mqm.Consumer("test_topic2", "c1")
				require.NoError(b.T(), err)
				c15, err := mqm.Consumer("test_topic2", "c1")
				require.NoError(b.T(), err)
				c16, err := mqm.Consumer("test_topic2", "c1")
				require.NoError(b.T(), err)
				return []mq.Consumer{
					c11,
					c12,
					c13,
					c14,
					c15,
					c16,
				}
			},
			consumeFunc: func(c mq.Consumer) []*mq.Message {
				msgCh, err := c.ConsumeChan(context.Background())
				require.NoError(b.T(), err)
				msgs := make([]*mq.Message, 0, 32)
				for val := range msgCh {
					msgs = append(msgs, val)
				}
				return msgs
			},
			wantVal: []*mq.Message{
				{
					Value: []byte("1"),
					Key:   []byte("1"),
					Topic: "test_topic2",
				},
				{
					Value: []byte("2"),
					Key:   []byte("2"),
					Topic: "test_topic2",
				},
				{
					Value: []byte("3"),
					Key:   []byte("3"),
					Topic: "test_topic2",
				},
				{
					Value: []byte("4"),
					Key:   []byte("4"),
					Topic: "test_topic2",
				},
				{
					Value: []byte("5"),
					Key:   []byte("5"),
					Topic: "test_topic2",
				},
			},
		},
	}
	for _, tc := range testcases {
		b.T().Run(tc.name, func(t *testing.T) {
			mqm := b.testMq
			err := mqm.Topic(context.Background(), tc.topic, int(tc.partitions))
			require.NoError(t, err)
			p, err := mqm.Producer(tc.topic)
			require.NoError(t, err)
			consumers := tc.consumers(mqm)
			ans := make([]*mq.Message, 0, len(tc.wantVal))
			var wg sync.WaitGroup
			locker := sync.RWMutex{}
			for _, c := range consumers {
				newc := c
				wg.Add(1)
				go func() {
					defer wg.Done()
					msgs := tc.consumeFunc(newc)
					locker.Lock()
					ans = append(ans, msgs...)
					locker.Unlock()
				}()
			}
			for _, msg := range tc.input {
				_, err := p.Produce(context.Background(), msg)
				require.NoError(t, err)
			}
			time.Sleep(5 * time.Second)
			err = closeConsumerAndProducer(consumers, []mq.Producer{p})
			require.NoError(t, err)
			wg.Wait()
			ansMsg := genMsg(ans, false)
			assert.ElementsMatch(t, tc.wantVal, ansMsg)
		})
	}
}

// 测试同一partition下的顺序
func (b *TestSuite) TestMQConsumer_OrderOfMessagesWithinAPartition() {
	testcases := []struct {
		name         string
		topic        string
		partitions   int64
		input        []*mq.Message
		consumers    func(mqm mq.MQ) []mq.Consumer
		consumerFunc func(c mq.Consumer) []*mq.Message
		wantVal      []*mq.Message
	}{
		{
			name:       "消息有序",
			topic:      "test_topic3",
			partitions: 3,
			input: []*mq.Message{
				{
					Key:   []byte("1"),
					Value: []byte("1"),
				},
				{
					Key:   []byte("1"),
					Value: []byte("2"),
				},
				{
					Key:   []byte("1"),
					Value: []byte("3"),
				},
				{
					Key:   []byte("1"),
					Value: []byte("4"),
				},
				{
					Key:   []byte("4"),
					Value: []byte("5"),
				},
				{
					Key:   []byte("4"),
					Value: []byte("6"),
				},
				{
					Key:   []byte("4"),
					Value: []byte("7"),
				},
				{
					Key:   []byte("4"),
					Value: []byte("8"),
				},
			},
			consumers: func(mqm mq.MQ) []mq.Consumer {
				c11, err := mqm.Consumer("test_topic3", "c1")
				require.NoError(b.T(), err)
				return []mq.Consumer{
					c11,
				}
			},
			consumerFunc: func(c mq.Consumer) []*mq.Message {
				msgCh, err := c.ConsumeChan(context.Background())
				require.NoError(b.T(), err)
				msgs := make([]*mq.Message, 0, 32)
				for val := range msgCh {
					msgs = append(msgs, val)
				}
				return msgs
			},
			wantVal: []*mq.Message{
				{
					Value: []byte("1"),
					Key:   []byte("1"),
					Topic: "test_topic3",
				},
				{
					Value: []byte("2"),
					Key:   []byte("1"),
					Topic: "test_topic3",
				},
				{
					Value: []byte("3"),
					Key:   []byte("1"),
					Topic: "test_topic3",
				},
				{
					Value: []byte("4"),
					Key:   []byte("1"),
					Topic: "test_topic3",
				},
				{
					Value: []byte("5"),
					Key:   []byte("4"),
					Topic: "test_topic3",
				},
				{
					Value: []byte("6"),
					Key:   []byte("4"),
					Topic: "test_topic3",
				},
				{
					Value: []byte("7"),
					Key:   []byte("4"),
					Topic: "test_topic3",
				},
				{
					Value: []byte("8"),
					Key:   []byte("4"),
					Topic: "test_topic3",
				},
			},
		},
	}
	for _, tc := range testcases {
		b.T().Run(tc.name, func(t *testing.T) {
			mqm := b.testMq
			err := mqm.Topic(context.Background(), tc.topic, int(tc.partitions))
			require.NoError(t, err)
			p, err := mqm.Producer(tc.topic)
			require.NoError(t, err)
			consumers := tc.consumers(mqm)
			ans := make([]*mq.Message, 0, len(tc.wantVal))
			var wg sync.WaitGroup
			locker := sync.RWMutex{}
			for _, c := range consumers {
				newc := c
				wg.Add(1)
				go func() {
					defer wg.Done()
					msgs := tc.consumerFunc(newc)
					locker.Lock()
					ans = append(ans, msgs...)
					locker.Unlock()
				}()
			}
			for _, msg := range tc.input {
				_, err := p.ProduceWithPartition(context.Background(), msg, 1)
				require.NoError(t, err)
			}

			time.Sleep(5 * time.Second)
			err = closeConsumerAndProducer(consumers, []mq.Producer{p})
			require.NoError(t, err)
			wg.Wait()
			wantMap := getMsgMap(tc.wantVal)
			actualMap := getMsgMap(genMsg(ans, false))
			assert.Equal(t, wantMap, actualMap)
		})
	}
}

// 测试发送到指定分区
func (b *TestSuite) TestMQProducer_ProduceWithSpecifiedPartitionID() {
	testcases := []struct {
		name         string
		topic        string
		partitions   int64
		input        []ProducerMsg
		consumerFunc func(c mq.Consumer) []*mq.Message
		wantVal      []*mq.Message
	}{
		{
			name:       "生产消息到指定分区",
			topic:      "test_topic4",
			partitions: 4,
			input: []ProducerMsg{
				{
					partition: 0,
					msg: &mq.Message{
						Key:   []byte("1"),
						Value: []byte("1"),
					},
				},
				{
					partition: 1,
					msg: &mq.Message{
						Key:   []byte("2"),
						Value: []byte("2"),
					},
				},
				{
					partition: 2,
					msg: &mq.Message{
						Key:   []byte("3"),
						Value: []byte("3"),
					},
				},
				{
					partition: 0,
					msg: &mq.Message{
						Key:   []byte("4"),
						Value: []byte("4"),
					},
				},
				{
					partition: 1,
					msg: &mq.Message{
						Key:   []byte("5"),
						Value: []byte("5"),
					},
				},
				{
					partition: 2,
					msg: &mq.Message{
						Key:   []byte("6"),
						Value: []byte("6"),
					},
				},
			},
			wantVal: []*mq.Message{
				{
					Value:       []byte("1"),
					Key:         []byte("1"),
					Topic:       "test_topic4",
					PartitionID: 0,
				},
				{
					Value:       []byte("2"),
					Key:         []byte("2"),
					Topic:       "test_topic4",
					PartitionID: 1,
				},
				{
					Value:       []byte("3"),
					Key:         []byte("3"),
					Topic:       "test_topic4",
					PartitionID: 2,
				},
				{
					Value:       []byte("4"),
					Key:         []byte("4"),
					Topic:       "test_topic4",
					PartitionID: 0,
				},
				{
					Value:       []byte("5"),
					Key:         []byte("5"),
					Topic:       "test_topic4",
					PartitionID: 1,
				},
				{
					Value:       []byte("6"),
					Key:         []byte("6"),
					Topic:       "test_topic4",
					PartitionID: 2,
				},
			},
			consumerFunc: func(c mq.Consumer) []*mq.Message {
				msgCh, err := c.ConsumeChan(context.Background())
				require.NoError(b.T(), err)
				msgs := make([]*mq.Message, 0, 32)
				for val := range msgCh {
					msgs = append(msgs, val)
				}
				return msgs
			},
		},
	}
	for _, tc := range testcases {
		b.T().Run(tc.name, func(t *testing.T) {
			mqm := b.testMq
			err := mqm.Topic(context.Background(), tc.topic, int(tc.partitions))
			require.NoError(t, err)
			p, err := mqm.Producer(tc.topic)
			require.NoError(t, err)
			c, err := mqm.Consumer(tc.topic, "1")
			require.NoError(t, err)
			consumers := []mq.Consumer{
				c,
			}
			ans := make([]*mq.Message, 0, len(tc.wantVal))
			var wg sync.WaitGroup
			locker := sync.RWMutex{}
			for _, c := range consumers {
				newc := c
				wg.Add(1)
				go func() {
					defer wg.Done()
					msgs := tc.consumerFunc(newc)
					locker.Lock()
					ans = append(ans, msgs...)
					locker.Unlock()
				}()
			}
			for _, msg := range tc.input {
				_, err := p.ProduceWithPartition(context.Background(), msg.msg, msg.partition)
				require.NoError(t, err)
			}
			time.Sleep(5 * time.Second)
			err = closeConsumerAndProducer([]mq.Consumer{c}, []mq.Producer{p})
			require.NoError(t, err)
			wg.Wait()
			assert.ElementsMatch(t, tc.wantVal, genWithPartitionMsg(ans))
		})
	}
}

// 测试producer调用close
func (b *TestSuite) TestMQProducer_Close() {
	t := b.T()
	topic := "test_topic5"
	mqm := b.testMq
	err := mqm.Topic(context.Background(), topic, 4)
	p, err := mqm.Producer(topic)
	require.NoError(b.T(), err)
	errChan := make(chan error, 10)
	// 开启三个goroutine使用 Produce
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := testProducer(p)
			if err != nil {
				errChan <- err
			}
		}()
	}
	// 开启三个goroutine使用 ProducerWithPartition
	// 0
	err = p.Close()
	require.NoError(t, err)
	wg.Wait()
	close(errChan)
	errList := make([]error, 0, len(errChan))
	for val := range errChan {
		errList = append(errList, val)
	}
	for _, e := range errList {
		require.True(t, errors.Is(e, mqerr.ErrProducerIsClosed))
	}
}

// 测试consumer调用close
func (b *TestSuite) TestMQAndConsumer_Close() {
	t := b.T()
	topic := "test_topic6"
	mqm := b.testMq
	err := mqm.Topic(context.Background(), topic, 4)
	require.NoError(t, err)
	c, err := mqm.Consumer(topic, "1")
	require.NoError(t, err)
	time.Sleep(3 * time.Second)
	// 调用close方法
	err = c.Close()
	require.NoError(t, err)
	// consumer会返回ErrConsumerIsClosed
	_, err = c.ConsumeChan(context.Background())
	assert.Equal(t, mqerr.ErrConsumerIsClosed, err)
	_, err = c.Consume(context.Background())
	assert.Equal(t, mqerr.ErrConsumerIsClosed, err)
	time.Sleep(3 * time.Second)
}

// 测试mq调用close
func (b *TestSuite) TestMQ_Close() {
	t := b.T()
	topic := "test_topic5"
	mqm := b.testMqCreator.Init()
	err := mqm.Topic(context.Background(), topic, 4)
	require.NoError(t, err)
	p, err := mqm.Producer(topic)
	require.NoError(t, err)
	c, err := mqm.Consumer(topic, "1")
	require.NoError(t, err)
	// 调用close方法
	err = mqm.Close()
	require.NoError(t, err)
	// mq会返回ErrMqIsClosed
	err = mqm.Topic(context.Background(), "test_topic6", 4)
	require.True(t, errors.Is(err, mqerr.ErrMQIsClosed))
	_, err = mqm.Producer(topic)
	require.True(t, errors.Is(err, mqerr.ErrMQIsClosed))
	_, err = mqm.Consumer(topic, "1")
	require.True(t, errors.Is(err, mqerr.ErrMQIsClosed))
	err = mqm.DeleteTopics(context.Background(), []string{topic})
	require.True(t, errors.Is(err, mqerr.ErrMQIsClosed))
	// producer会返回ErrProducerIsClosed
	_, err = p.Produce(context.Background(), &mq.Message{})
	require.True(t, errors.Is(err, mqerr.ErrProducerIsClosed))
	_, err = p.ProduceWithPartition(context.Background(), &mq.Message{}, 0)
	require.True(t, errors.Is(err, mqerr.ErrProducerIsClosed))
	// consumer会返回ErrConsumerIsClosed
	_, err = c.ConsumeChan(context.Background())
	require.True(t, errors.Is(err, mqerr.ErrConsumerIsClosed))
	_, err = c.Consume(context.Background())
	require.True(t, errors.Is(err, mqerr.ErrConsumerIsClosed))
}

func genMsg(msgs []*mq.Message, hasPartitionID bool) []*mq.Message {
	for index := range msgs {
		if !hasPartitionID {
			msgs[index].PartitionID = 0
		}
		msgs[index].Offset = 0
		msgs[index].Header = nil
	}
	return msgs
}

func genWithPartitionMsg(msgs []*mq.Message) []*mq.Message {
	for index := range msgs {
		msgs[index].Offset = 0
		msgs[index].Header = nil
	}
	return msgs
}

func getMsgMap(msgs []*mq.Message) map[string][]*mq.Message {
	wantMap := make(map[string][]*mq.Message, 10)
	for _, val := range msgs {
		_, ok := wantMap[string(val.Key)]
		if !ok {
			wantMap[string(val.Key)] = append([]*mq.Message{}, val)
		} else {
			wantMap[string(val.Key)] = append(wantMap[string(val.Key)], val)
		}
	}
	return wantMap
}

func testProducer(p mq.Producer) error {
	for {
		_, err := p.Produce(context.Background(), &mq.Message{
			Value: []byte("1"),
		})
		if err != nil {
			return err
		}
	}
}

func closeConsumerAndProducer(consumers []mq.Consumer, producers []mq.Producer) error {
	errList := make([]error, 0, len(consumers)+len(producers))
	for _, c := range consumers {
		err := c.Close()
		if err != nil {
			errList = append(errList, err)
		}
	}
	for _, p := range producers {
		err := p.Close()
		if err != nil {
			errList = append(errList, err)
		}
	}
	return multierr.Combine(errList...)
}