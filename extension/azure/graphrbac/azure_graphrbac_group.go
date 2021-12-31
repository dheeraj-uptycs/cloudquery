package graphrbac

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/Uptycs/cloudquery/extension/azure"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/cloudquery/utilities"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/fatih/structs"
)

const azureGraphrbacGroup string = "azure_graphrbac_group"

// GraphrbacGroupColunmns returns the list of columns in the table
func GraphrbacGroupColunmns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("deletion_timestamp"),
		table.TextColumn("display_name"),
		table.TextColumn("mail"),
		table.TextColumn("mail_enabled"),
		table.TextColumn("mail_nickname"),
		table.TextColumn("object_id"),
		table.TextColumn("object_type"),
		table.TextColumn("security_enabled"),
	}
}

// GraphrbacGroupGenerate returns the rows in the table for all configured accounts
func GraphrbacGroupGenerate(osqCtx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	resultMap := make([]map[string]string, 0)
	if len(utilities.ExtConfiguration.ExtConfAzure.Accounts) == 0 {
		utilities.GetLogger().WithFields(log.Fields{
			"tableName": azureGraphrbacGroup,
			"account":   "default",
		}).Info("processing account")
		results, err := processAccountGraphrbacGroup(nil)
		if err != nil {
			return resultMap, err
		}
		resultMap = append(resultMap, results...)
	} else {
		for _, account := range utilities.ExtConfiguration.ExtConfAzure.Accounts {
			utilities.GetLogger().WithFields(log.Fields{
				"tableName": azureGraphrbacGroup,
				"account":   account.SubscriptionID,
			}).Info("processing account")
			results, err := processAccountGraphrbacGroup(&account)
			if err != nil {
				continue
			}
			resultMap = append(resultMap, results...)
		}
	}

	return resultMap, nil
}

func processAccountGraphrbacGroup(account *utilities.ExtensionConfigurationAzureAccount) ([]map[string]string, error) {
	resultMap := make([]map[string]string, 0)

	session, err := azure.GetAuthSession(account)
	if err != nil {
		return resultMap, err
	}

	tableConfig, ok := utilities.TableConfigurationMap[azureGraphrbacGroup]
	if !ok {
		utilities.GetLogger().WithFields(log.Fields{
			"tableName": azureGraphrbacGroup,
		}).Error("failed to get table configuration")
		return resultMap, fmt.Errorf("table configuration not found")
	}

	setGraphrbacGrouptoTable(account.TenantID, session, &resultMap, tableConfig)

	return resultMap, nil
}

func setGraphrbacGrouptoTable(tenantId string, session *azure.AzureSession, resultMap *[]map[string]string, tableConfig *utilities.TableConfig) {

	for resourcesItr, err := getGraphrbacGroupData(session, tenantId); resourcesItr.NotDone(); resourcesItr.Next() {
		if err != nil {
			utilities.GetLogger().WithFields(log.Fields{
				"tableName": azureGraphrbacGroup,
				"TenantId":  tenantId,
				"errString": err.Error(),
			}).Error("failed to get group list iterator zones")
		}

		resource := resourcesItr.Value()

		structs.DefaultTagName = "json"
		resMap := structs.Map(resource)
		byteArr, err := json.Marshal(resMap)
		if err != nil {
			utilities.GetLogger().WithFields(log.Fields{
				"tableName": azureGraphrbacGroup,
				"TenantId":  tenantId,
				"errString": err.Error(),
			}).Error("failed to marshal response")
			continue
		}
		table := utilities.NewTable(byteArr, tableConfig)
		for _, row := range table.Rows {
			result := azure.RowToMap(row, session.SubscriptionId, "", "", tableConfig)
			*resultMap = append(*resultMap, result)
		}
	}

}
func getGraphrbacGroupData(session *azure.AzureSession, tenantId string) (result graphrbac.GroupListResultIterator, err error) {
	svcClient := graphrbac.NewGroupsClient(tenantId)
	svcClient.Authorizer = session.GraphAuthorizer
	return svcClient.ListComplete(context.Background(), "")
}
