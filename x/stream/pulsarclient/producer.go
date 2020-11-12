package pulsarclient

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/okex/okexchain/x/stream/common"
	"strconv"
	"sync"
	"time"

	"github.com/Comcast/pulsar-client-go"
	appCfg "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/google/uuid"
	"github.com/okex/okexchain/x/backend"
	"github.com/tendermint/tendermint/libs/log"
)

type PulsarProducer struct {
	producers                     []*pulsar.ManagedProducer
	partion                       int64
	marketServiceEnable           bool
	marketEurekaURL               string
	marketEurekaRegisteredAppName string
}

func NewPulsarProducer(url string, cfg *appCfg.StreamConfig, logger log.Logger, asyncErrs *chan error) *PulsarProducer {
	var mp = &PulsarProducer{producers: make([]*pulsar.ManagedProducer, 0, cfg.MarketPulsarPartition),
		partion: int64(cfg.MarketPulsarPartition), marketServiceEnable: cfg.MarketServiceEnable, marketEurekaURL: cfg.EurekaServerUrl,
		marketEurekaRegisteredAppName: cfg.MarketQuotationsEurekaName}

	for i := 0; i < cfg.MarketPulsarPartition; i++ {
		mcp := pulsar.NewManagedClientPool()
		mpCfg := pulsar.ManagedProducerConfig{
			Name:                  uuid.New().String() + "-subs_standard_dex_spot-" + strconv.Itoa(i),
			Topic:                 cfg.MarketPulsarTopic + "-partition-" + strconv.Itoa(i),
			NewProducerTimeout:    time.Second * 3,
			InitialReconnectDelay: time.Second,
			MaxReconnectDelay:     time.Minute,
			ManagedClientConfig: pulsar.ManagedClientConfig{
				ClientConfig: pulsar.ClientConfig{
					Addr: url,
					Errs: *asyncErrs,
				},
			},
		}
		mp.producers = append(mp.producers, pulsar.NewManagedProducer(mcp, mpCfg))
		logger.Info(fmt.Sprintf("%s try to create producer on topic %s on url:%s", mpCfg.Name, mpCfg.Topic, url))
	}
	return mp
}

func (p *PulsarProducer) RefreshMarketIDMap(data *common.KlineData, logger log.Logger) error {
	logger.Debug(
		fmt.Sprintf("marketServiceEnable:%v, eurekaUrl:%s, registerAppName:%s",
			p.marketServiceEnable, p.marketEurekaURL, p.marketEurekaRegisteredAppName),
	)
	for _, tokenPair := range data.GetNewTokenPairs() {
		tokenPairName := tokenPair.Name()
		marketIDMap := common.GetMarketIDMap()
		marketIDMap[tokenPairName] = int64(tokenPair.ID)
		logger.Debug(fmt.Sprintf("set new tokenpair %+v in map, MarketIdMap: %+v", tokenPair, marketIDMap))

		if p.marketServiceEnable {
			marketServiceURL, err := common.GetMarketServiceURL(p.marketEurekaURL, p.marketEurekaRegisteredAppName)
			if err == nil {
				logger.Debug(fmt.Sprintf("successfully get the market service url [%s]", marketServiceURL))
			} else {
				logger.Error(fmt.Sprintf("failed to get the market service url [%s]. error: %s", marketServiceURL, err))
			}

			err = common.RegisterNewTokenPair(int64(tokenPair.ID), tokenPairName, marketServiceURL, logger)
			if err != nil {
				logger.Error(fmt.Sprintf("failed register tokenpair %+v in market service. error: %s", tokenPair, err))
				return err
			}
		}
	}
	return nil
}

func (p *PulsarProducer) SendAllMsg(data *common.KlineData, logger log.Logger) (map[string]int, error) {
	// log := logger.With("module", "pulsar")
	result := make(map[string]int)
	matchResults := data.GetMatchResults()
	result["matchResults"] = len(matchResults)
	if len(matchResults) == 0 {
		return result, nil
	}

	var errChan = make(chan error, len(matchResults))
	var wg sync.WaitGroup
	wg.Add(len(matchResults))
	for _, matchResult := range matchResults {
		go func(matchResult backend.MatchResult) {
			defer wg.Done()
			marketID, ok := common.GetMarketIDMap()[matchResult.Product]
			if !ok {
				err := fmt.Errorf("failed to find %s marketId", matchResult.Product)
				errChan <- err
				return
			}

			msg, err := json.Marshal(&matchResult)
			if err != nil {
				errChan <- err
				return
			}

			sctx, cancle := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancle()
			_, err = p.producers[marketID%(p.partion)].Send(sctx, msg)
			if err != nil {
				errChan <- err
				return
			}

			logger.Debug(
				fmt.Sprintf("successfully send matchResult [marketId:%d, CreatedTime:%s, BlockHeight:%d, Quantity:%f, Price:%f, InstrumentName:%s]",
					marketID, time.Unix(matchResult.Timestamp, 0).Format("2006-01-02 15:04:05"), matchResult.BlockHeight,
					matchResult.Quantity, matchResult.Price, matchResult.Product,
				),
			)
		}(*matchResult)
	}
	wg.Wait()

	if len(errChan) != 0 {
		err := <-errChan
		return result, err
	}
	return result, nil
}