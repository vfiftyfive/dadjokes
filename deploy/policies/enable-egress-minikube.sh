helm upgrade --install otterize otterize/otterize-kubernetes -n otterize-system --create-namespace \
	--set global.otterizeCloud.credentials.clientId=cli_jjpekf4jef \
	--set global.otterizeCloud.credentials.clientSecret=32f8759a4ef4c0de07c13667827d93939b88ea11c22abf1611c2ad81bd167dfb \
	--set intentsOperator.operator.mode=defaultActive \
	--set intentsOperator.operator.enableEgressNetworkPolicyCreation=true \
	--set networkMapper.dnsClientIntentsUpdateEnabled=true
