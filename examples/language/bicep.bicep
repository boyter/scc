/*
this is a multi
line comment
*/
param rgLocation string = resourceGroup().location
param storageNames array = [
  'contoso'
  'fabrikam'
]

// this is a comment
resource createStorages 'Microsoft.Storage/storageAccounts@2022-09-01' = [for name in storageNames: {
  name: '${name}str${uniqueString(resourceGroup().id)}'
  location: rgLocation
  sku: {
    name: 'Standard_LRS'
  }
  kind: '''StorageV2'''
}]

param storageAccountName string
param location string = resourceGroup().location

@allowed([
  'new'
  'existing'
])
param newOrExisting string = 'new'

resource saNew 'Microsoft.Storage/storageAccounts@2022-09-01' = if (newOrExisting == 'new') {
  name: storageAccountName
  location: location
  sku: {
    name: 'Standard_LRS'
  }
  kind: 'StorageV2'
}

resource saExisting 'Microsoft.Storage/storageAccounts@2022-09-01' existing = if (newOrExisting == 'existing') {
  name: storageAccountName
}

output storageAccountId string = ((newOrExisting == 'new') ? saNew.id : saExisting.id)

param deployZone bool

resource dnsZone 'Microsoft.Network/dnszones@2018-05-01' = if (deployZone) {
  name: 'myZone'
  location: 'global'
}
