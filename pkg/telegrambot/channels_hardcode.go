package telegrambot

var ChannelIDs = map[EnvType][]ChannelInfo{
	EnvTypeDev: {
		{
			ChatID:    TestChatID,
			BlockSlug: "2ngt",
		},
		{
			ChatID:    TestChatID,
			BlockSlug: "ytnv",
		},
	},
	EnvTypeTesting: {
		{
			ChatID:    TestChatID,
			BlockSlug: "2ngt",
		},
		{
			ChatID:    TestChatID,
			BlockSlug: "ytnv",
		},
	},
	EnvTypeProd: {
		{
			ChatID:    -1001451631453,
			BlockSlug: "2ngt",
		},
		{
			ChatID:    -1001439896663,
			BlockSlug: "sp",
		},
		{
			ChatID:    -1002066659264,
			BlockSlug: "ytnv",
		},
		{
			ChatID:    -1002087536270,
			BlockSlug: "kolskaya8",
		},
		{
			ChatID:    -1002123708132,
			BlockSlug: "hp",
		},
	},
}
