package types

var (
	// registeredApplications represents the list of applications present in smart-contract.
	//
	// NOTE: Now here's only values for testing purposes. Final list will be set up on release.
	registeredApplications = []ApplicationListItem{
		{0, "haqq15gl76py2lqqrlawzs0afkmh9k7kxc6wmvcqqlm", "haqq15gl76py2lqqrlawzs0afkmh9k7kxc6wmvcqqlm", 0, "123000000000000000000000", "0"},
		{1, "haqq15gl76py2lqqrlawzs0afkmh9k7kxc6wmvcqqlm", "haqq15gl76py2lqqrlawzs0afkmh9k7kxc6wmvcqqlm", 1, "456000000000000000000000", "123000000000000000000000"},
		{2, "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", 0, "789000000000000000000000", "579000000000000000000000"},
		{3, "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", "haqq19pxv2r4key79twjfv0gdc5yhc4xmw9vqxkj2nl", 1, "212300000000000000000000", "1368000000000000000000000"},
		{4, "haqq1m2q0lsn655c7gxudz3srepgpm5sfv0dfwuje52", "haqq1m2q0lsn655c7gxudz3srepgpm5sfv0dfwuje52", 0, "345600000000000000000000", "1580300000000000000000000"},
		{5, "haqq1m2q0lsn655c7gxudz3srepgpm5sfv0dfwuje52", "haqq1m2q0lsn655c7gxudz3srepgpm5sfv0dfwuje52", 1, "478900000000000000000000", "1925900000000000000000000"},
		{6, "haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6", "haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6", 0, "512300000000000000000000", "2404800000000000000000000"},
		{7, "haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6", "haqq1s873eyxj6v6k5ycut3sd00uvuf3kw8xfa2nql6", 1, "645600000000000000000000", "2917100000000000000000000"},
		{8, "haqq10dxuq79zws47lvg5vsj58mvsft86l4um8xsnen", "haqq10dxuq79zws47lvg5vsj58mvsft86l4um8xsnen", 0, "778900000000000000000000", "3562700000000000000000000"},
		{9, "haqq10dxuq79zws47lvg5vsj58mvsft86l4um8xsnen", "haqq10dxuq79zws47lvg5vsj58mvsft86l4um8xsnen", 1, "800123000000000000000000", "4341600000000000000000000"},
		{10, "haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp", "haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp", 0, "900456000000000000000000", "5141723000000000000000000"},
		{11, "haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp", "haqq1wk0qkkzxrs9262pzu5mgm4zv4a3qy8kxqqa7cp", 1, "10000000000000000000000000", "6042179000000000000000000"},
		{12, "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", 0, "15000000000000000000000000", "16042179000000000000000000"},
		{13, "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", "haqq1gcwaegn8j68t02qersxdk95y53dey5xfq9alp0", 1, "20000000000000000000000000", "31042179000000000000000000"},
	}
	// 31042179 + 20000000 = 51,042,179
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
