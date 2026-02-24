package types

var (
	// registeredApplications represents the list of applications present in smart-contract.
	//
	// NOTE: Now here's only values for testing purposes. Final list will be set up on release.
	registeredApplications = []ApplicationListItem{
		{0, "haqq15gl76py2lqqrlawzs0afkmh9k7kxc6wmvcqqlm", "haqq15gl76py2lqqrlawzs0afkmh9k7kxc6wmvcqqlm", 1, "123000000000000000000000", "0"},
		{1, "haqq15gl76py2lqqrlawzs0afkmh9k7kxc6wmvcqqlm", "haqq15gl76py2lqqrlawzs0afkmh9k7kxc6wmvcqqlm", 2, "456000000000000000000000", "123000000000000000000000"},
		{2, "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", 1, "789000000000000000000000", "579000000000000000000000"},
		{3, "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", 2, "2000123000000000000000000", "1368000000000000000000000"},
		{4, "haqq1m2q0lsn655c7gxudz3srepgpm5sfv0dfwuje52", "haqq1m2q0lsn655c7gxudz3srepgpm5sfv0dfwuje52", 1, "3000456000000000000000000", "3368123000000000000000000"},
		{5, "haqq1m2q0lsn655c7gxudz3srepgpm5sfv0dfwuje52", "haqq1m2q0lsn655c7gxudz3srepgpm5sfv0dfwuje52", 2, "4000789000000000000000000", "6368579000000000000000000"},
		{6, "haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6", "haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6", 1, "5123000000000000000000000", "10369368000000000000000000"},
		{7, "haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6", "haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6", 2, "6456000000000000000000000", "15492368000000000000000000"},
		{8, "haqq10dxuq79zws47lvg5vsj58mvsft86l4um8xsnen", "haqq10dxuq79zws47lvg5vsj58mvsft86l4um8xsnen", 1, "7789000000000000000000000", "21948368000000000000000000"},
		{9, "haqq10dxuq79zws47lvg5vsj58mvsft86l4um8xsnen", "haqq10dxuq79zws47lvg5vsj58mvsft86l4um8xsnen", 2, "80012300000000000000000000", "29737368000000000000000000"},
		{10, "haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp", "haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp", 1, "90045600000000000000000000", "109749668000000000000000000"},
		{11, "haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp", "haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp", 2, "100000000000000000000000000", "199795268000000000000000000"},
		{12, "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", 1, "200000000000000000000000000", "299795268000000000000000000"},
		{13, "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", 2, "300000000000000000000000000", "499795268000000000000000000"},
	}

	// registeredApplicationsBySender is a helper index of applications grouped by sender address.
	registeredApplicationsBySender = map[string][]uint64{
		"haqq15gl76py2lqqrlawzs0afkmh9k7kxc6wmvcqqlm": {0, 1},
		"haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl": {2, 3},
		"haqq1m2q0lsn655c7gxudz3srepgpm5sfv0dfwuje52": {4, 5},
		"haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6": {6, 7},
		"haqq10dxuq79zws47lvg5vsj58mvsft86l4um8xsnen": {8, 9},
		"haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp": {10, 11},
		"haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0": {12, 13},
	}
)
