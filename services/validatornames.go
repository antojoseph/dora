package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethpandaops/dora/config"
	"github.com/ethpandaops/dora/db"
	"github.com/ethpandaops/dora/dbtypes"
	"github.com/ethpandaops/dora/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
)

var logger_vn = logrus.StandardLogger().WithField("module", "validator_names")

type ValidatorNames struct {
	loadingMutex      sync.Mutex
	loading           bool
	namesMutex        sync.RWMutex
	namesByIndex      map[uint64]string
	namesByWithdrawal map[common.Address]string
}

func (vn *ValidatorNames) GetValidatorName(index uint64) string {
	if !vn.namesMutex.TryRLock() {
		return ""
	}
	defer vn.namesMutex.RUnlock()
	if vn.namesByIndex == nil {
		return ""
	}

	name := vn.namesByIndex[index]
	if name != "" {
		return name
	}

	validatorSet := GlobalBeaconService.GetCachedValidatorSet()
	if validatorSet == nil {
		return ""
	}

	validator := validatorSet[phase0.ValidatorIndex(index)]
	if validator == nil {
		return ""
	}

	if validator.Validator.WithdrawalCredentials[0] == 0x01 {
		withdrawal := common.Address(validator.Validator.WithdrawalCredentials[12:])
		name = vn.namesByWithdrawal[withdrawal]
		if name != "" {
			return name
		}
	}

	return ""
}

func (vn *ValidatorNames) GetValidatorNamesCount() uint64 {
	if !vn.namesMutex.TryRLock() {
		return 0
	}
	defer vn.namesMutex.RUnlock()
	if vn.namesByIndex == nil {
		return 0
	}
	return uint64(len(maps.Keys(vn.namesByIndex)) + len(maps.Keys(vn.namesByWithdrawal)))
}

func (vn *ValidatorNames) LoadValidatorNames() {
	vn.loadingMutex.Lock()
	defer vn.loadingMutex.Unlock()
	if vn.loading {
		return
	}
	vn.loading = true

	go func() {
		vn.namesMutex.Lock()
		vn.namesByIndex = make(map[uint64]string)
		vn.namesByWithdrawal = make(map[common.Address]string)
		vn.namesMutex.Unlock()

		// load names
		if strings.HasPrefix(utils.Config.Frontend.ValidatorNamesYaml, "~internal/") {
			err := vn.loadFromInternalYaml(utils.Config.Frontend.ValidatorNamesYaml[10:])
			if err != nil {
				logger_vn.WithError(err).Errorf("error while loading validator names from internal yaml")
			}
		} else if utils.Config.Frontend.ValidatorNamesYaml != "" {
			err := vn.loadFromYaml(utils.Config.Frontend.ValidatorNamesYaml)
			if err != nil {
				logger_vn.WithError(err).Errorf("error while loading validator names from yaml")
			}
		}
		if utils.Config.Frontend.ValidatorNamesInventory != "" {
			err := vn.loadFromRangesApi(utils.Config.Frontend.ValidatorNamesInventory)
			if err != nil {
				logger_vn.WithError(err).Errorf("error while loading validator names inventory")
			}
		}

		// update db
		if !utils.Config.Indexer.DisableIndexWriter {
			vn.updateDb()
		}

		vn.loading = false
	}()
}

func (vn *ValidatorNames) loadFromYaml(fileName string) error {
	f, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("error opening validator names file %v: %v", fileName, err)
	}
	defer f.Close()

	namesYaml := map[string]string{}
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&namesYaml)
	if err != nil {
		return fmt.Errorf("error decoding validator names file %v: %v", fileName, err)
	}

	nameCount := vn.parseNamesMap(namesYaml)
	logger_vn.Infof("loaded %v validator names from yaml (%v)", nameCount, fileName)

	return nil
}

func (vn *ValidatorNames) loadFromInternalYaml(fileName string) error {
	f, err := config.ValidatorNamesYml.Open(fileName)
	if err != nil {
		return fmt.Errorf("could not find internal validator names file %v: %v", fileName, err)
	}

	namesYaml := map[string]string{}
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&namesYaml)
	if err != nil {
		return fmt.Errorf("could not find internal validator names file %v: %v", fileName, err)
	}

	nameCount := vn.parseNamesMap(namesYaml)
	logger_vn.Infof("loaded %v validator names from internal yaml (%v)", nameCount, fileName)

	return nil
}

func (vn *ValidatorNames) parseNamesMap(names map[string]string) int {
	vn.namesMutex.Lock()
	defer vn.namesMutex.Unlock()
	nameCount := 0
	for idxStr, name := range names {
		rangeParts := strings.Split(idxStr, ":")
		if len(rangeParts) > 1 {
			switch rangeParts[0] {
			case "withdrawal":
				withdrawal := common.HexToAddress(rangeParts[1])
				vn.namesByWithdrawal[withdrawal] = name
				nameCount++
			}

		} else {
			rangeParts = strings.Split(idxStr, "-")
			minIdx, err := strconv.ParseUint(rangeParts[0], 10, 64)
			if err != nil {
				continue
			}
			maxIdx := minIdx + 1
			if len(rangeParts) > 1 {
				maxIdx, err = strconv.ParseUint(rangeParts[1], 10, 64)
				if err != nil {
					continue
				}
			}
			for idx := minIdx; idx <= maxIdx; idx++ {
				vn.namesByIndex[idx] = name
				nameCount++
			}
		}
	}
	return nameCount
}

type validatorNamesRangesResponse struct {
	Ranges map[string]string `json:"ranges"`
}

func (vn *ValidatorNames) loadFromRangesApi(apiUrl string) error {
	logger_vn.Debugf("Loading validator names from inventory: %v", apiUrl)

	client := &http.Client{Timeout: time.Second * 120}
	resp, err := client.Get(apiUrl)
	if err != nil {
		return fmt.Errorf("could not fetch inventory (%v): %v", utils.GetRedactedUrl(apiUrl), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			logger_vn.Errorf("could not fetch inventory (%v): not found", utils.GetRedactedUrl(apiUrl))
			return nil
		}
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("url: %v, error-response: %s", utils.GetRedactedUrl(apiUrl), data)
	}
	rangesResponse := &validatorNamesRangesResponse{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&rangesResponse)
	if err != nil {
		return fmt.Errorf("error parsing validator ranges response: %v", err)
	}

	nameCount := vn.parseNamesMap(rangesResponse.Ranges)
	logger_vn.Infof("loaded %v validator names from inventory api (%v)", nameCount, utils.GetRedactedUrl(apiUrl))
	return nil
}

func (vn *ValidatorNames) updateDb() error {
	vn.namesMutex.RLock()
	nameRows := make([]*dbtypes.ValidatorName, 0)
	for index, name := range vn.namesByIndex {
		nameRows = append(nameRows, &dbtypes.ValidatorName{
			Index: index,
			Name:  name,
		})
	}
	vn.namesMutex.RUnlock()

	sort.Slice(nameRows, func(a, b int) bool {
		return nameRows[a].Index < nameRows[b].Index
	})

	tx, err := db.WriterDb.Beginx()
	if err != nil {
		return fmt.Errorf("error starting db transaction: %v", err)
	}
	defer tx.Rollback()

	batchSize := 10000

	lastIndex := uint64(0)
	nameIdx := 0
	nameLen := len(nameRows)
	for nameIdx < nameLen {
		maxIdx := nameIdx + batchSize
		if maxIdx >= nameLen {
			maxIdx = nameLen
		}
		sliceLen := maxIdx - nameIdx
		namesSlice := nameRows[nameIdx:maxIdx]
		maxIndex := namesSlice[sliceLen-1].Index

		// get existing db entries
		dbNamesMap := map[uint64]string{}
		for _, dbName := range db.GetValidatorNames(lastIndex, maxIndex, tx) {
			dbNamesMap[dbName.Index] = dbName.Name
		}

		// get diffs
		updateNames := make([]*dbtypes.ValidatorName, 0)
		for _, nameRow := range namesSlice {
			dbName := dbNamesMap[nameRow.Index]
			delete(dbNamesMap, nameRow.Index)
			if dbName == nameRow.Name {
				continue // no update
			}
			updateNames = append(updateNames, nameRow)
		}

		removeIndexes := make([]uint64, 0)
		for index := range dbNamesMap {
			removeIndexes = append(removeIndexes, index)
		}

		if len(updateNames) > 0 {
			err := db.InsertValidatorNames(updateNames, tx)
			if err != nil {
				logger_vn.WithError(err).Errorf("error while adding validator names to db")
			}
		}
		if len(removeIndexes) > 0 {
			err := db.DeleteValidatorNames(removeIndexes, tx)
			if err != nil {
				logger_vn.WithError(err).Errorf("error while deleting validator names from db")
			}
		}
		logger_vn.Debugf("update validator names %v-%v: %v changed, %v removed", lastIndex, maxIdx, len(updateNames), len(removeIndexes))

		lastIndex = maxIndex + 1
		nameIdx = maxIdx
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing db transaction: %v", err)
	}

	return nil
}
