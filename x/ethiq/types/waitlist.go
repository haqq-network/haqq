package types

var (
	// registeredApplications represents the list of applications present in smart-contract.
	//
	// NOTE: Now here's only values for testing purposes. Final list will be set up on release.
	registeredApplications = []ApplicationListItem{
		{0, "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", 0, "1000000000000000000", "0"},
		{1, "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", 0, "10000000000000000000", "1000000000000000000"},
		{2, "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", 0, "1000000000000000000", "11000000000000000000"},
		{3, "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", 0, "2000000000000000000", "12000000000000000000"},
		{4, "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", 0, "20000000000000000000", "14000000000000000000"},
		{5, "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", 0, "3000000000000000000", "34000000000000000000"},
		{6, "haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z", "haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z", 0, "100000000000000000000", "37000000000000000000"},
		{7, "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", 0, "0", "137000000000000000000"},
		{8, "haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq", "haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq", 0, "200000000000000000000", "137000000000000000000"},
		{9, "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", 0, "2980330604876500000000", "337000000000000000000"},
		{10, "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", 0, "0", "3317330604876500000000"},
		{11, "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", 0, "0", "3317330604876500000000"},
		{12, "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", 1, "1000000000000000000000000", "3317330604876500000000"},
		{13, "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", 0, "0", "1003317330604876500000000"},
		{14, "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", 0, "0", "1003317330604876500000000"},
		{15, "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", 0, "0", "1003317330604876500000000"},
		{16, "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", 0, "0", "1003317330604876500000000"},
		{17, "haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z", "haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z", 0, "1988243498905143433831", "1003317330604876500000000"},
		{18, "haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z", "haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z", 1, "11000000000000000000", "1005305574103781643433831"},
		{19, "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", 0, "0", "1005316574103781643433831"},
		{20, "haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z", "haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z", 1, "5000000000000000000", "1005316574103781643433831"},
		{21, "haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp", "haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp", 0, "1000000000000000000", "1005321574103781643433831"},
		{22, "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", 1, "100000000000000000000000000", "1005322574103781643433831"},
		{23, "haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp", "haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp", 0, "1000000000000000000", "101005322574103781643433831"},
		{24, "haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z", "haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z", 0, "1000000000000000000", "101005323574103781643433831"},
		{25, "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", 0, "0", "101005324574103781643433831"},
		{26, "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", 0, "0", "101005324574103781643433831"},
		{27, "haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq", "haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq", 0, "10000000000000000000", "101005324574103781643433831"},
		{28, "haqq1m2q0lsn655c7gxudz3srepgpm5sfv0dfwuje52", "haqq1m2q0lsn655c7gxudz3srepgpm5sfv0dfwuje52", 0, "10000000000000000000", "101005334574103781643433831"},
		{29, "haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq", "haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq", 0, "1000000000000000000", "101005344574103781643433831"},
		{30, "haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq", "haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq", 0, "9551452743667990266", "101005345574103781643433831"},
		{31, "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", 0, "10000000000000000000000", "101005355125556525311424097"},
		{32, "haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq", "haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq", 0, "554982369554263917510", "101015355125556525311424097"},
		{33, "haqq1399phzv4ke3gzvwr2e8jdtal0zsh4a6a2j89t2", "haqq1399phzv4ke3gzvwr2e8jdtal0zsh4a6a2j89t2", 0, "1000000000000000000", "101015910107926079575341607"},
		{34, "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", 0, "0", "101015911107926079575341607"},
		{35, "haqq1wxytrgv06zdquatkp9g6ukrhawla6upxt9lttz", "haqq1wxytrgv06zdquatkp9g6ukrhawla6upxt9lttz", 0, "452615777340377913632642", "101015911107926079575341607"},
		{36, "haqq1wxytrgv06zdquatkp9g6ukrhawla6upxt9lttz", "haqq1wxytrgv06zdquatkp9g6ukrhawla6upxt9lttz", 0, "68000000000000000000000", "101468526885266457488974249"},
		{37, "haqq1m2q0lsn655c7gxudz3srepgpm5sfv0dfwuje52", "haqq1m2q0lsn655c7gxudz3srepgpm5sfv0dfwuje52", 0, "1000000000000000000000", "101536526885266457488974249"},
		{38, "haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6", "haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6", 0, "110000000000000000000", "101537526885266457488974249"},
		{39, "haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6", "haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6", 0, "634000000000000000000", "101537636885266457488974249"},
		{40, "haqq1rt5ewr3un923xn83cs5q8njgl2wxvyq4kszepf", "haqq1rt5ewr3un923xn83cs5q8njgl2wxvyq4kszepf", 0, "90000000000000000000", "101538270885266457488974249"},
		{41, "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", 0, "0", "101538360885266457488974249"},
		{42, "haqq1rt5ewr3un923xn83cs5q8njgl2wxvyq4kszepf", "haqq1rt5ewr3un923xn83cs5q8njgl2wxvyq4kszepf", 0, "46000000000000000000", "101538360885266457488974249"},
		{43, "haqq1twl2zfe76n0dzddh9tcle7a9rqcmvfm43elxvm", "haqq1twl2zfe76n0dzddh9tcle7a9rqcmvfm43elxvm", 0, "100000000000000000000000", "101538406885266457488974249"},
		{44, "haqq1twl2zfe76n0dzddh9tcle7a9rqcmvfm43elxvm", "haqq1twl2zfe76n0dzddh9tcle7a9rqcmvfm43elxvm", 0, "1000000000000000000000000", "101638406885266457488974249"},
		{45, "haqq1twl2zfe76n0dzddh9tcle7a9rqcmvfm43elxvm", "haqq1twl2zfe76n0dzddh9tcle7a9rqcmvfm43elxvm", 0, "20000000000000000000000000", "102638406885266457488974249"},
		{46, "haqq1twl2zfe76n0dzddh9tcle7a9rqcmvfm43elxvm", "haqq1twl2zfe76n0dzddh9tcle7a9rqcmvfm43elxvm", 0, "500000000000000000000", "122638406885266457488974249"},
		{47, "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", "haqq1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq3y4vet", 0, "0", "122638906885266457488974249"},
		{48, "haqq1twl2zfe76n0dzddh9tcle7a9rqcmvfm43elxvm", "haqq1twl2zfe76n0dzddh9tcle7a9rqcmvfm43elxvm", 0, "1000000000000000000000000", "122638906885266457488974249"},
	}
	// 122638906885266457488974249 + 1000000000000000000000000 = 123,638,906.885266457488974249 ISLM

	// registeredApplicationsBySender is a helper index of applications grouped by sender address.
	registeredApplicationsBySender = map[string][]uint64{
		"haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl": {0, 1, 31},
		"haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0": {2, 3, 4, 5, 9, 12, 22},
		"haqq1hexv24j6g035fzwj7qhdj6qtgdh9jcurkug09z": {6, 17, 18, 20, 24},
		"haqq1f3u5gz9fj2v3sxf7j9szsl2c7mfmcae2m6lslq": {8, 27, 29, 30, 32},
		"haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp": {21, 23},
		"haqq1m2q0lsn655c7gxudz3srepgpm5sfv0dfwuje52": {28, 37},
		"haqq1399phzv4ke3gzvwr2e8jdtal0zsh4a6a2j89t2": {33},
		"haqq1wxytrgv06zdquatkp9g6ukrhawla6upxt9lttz": {35, 36},
		"haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6": {38, 39},
		"haqq1rt5ewr3un923xn83cs5q8njgl2wxvyq4kszepf": {40, 42},
		"haqq1twl2zfe76n0dzddh9tcle7a9rqcmvfm43elxvm": {43, 44, 45, 46, 48},
	}
)
