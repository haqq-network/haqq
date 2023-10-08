package v160

import (
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
)

type accountData struct {
	accNum uint64
	seq    uint64
}

var accounts = map[string]accountData{
	"haqq1qqqn3fdcatuy5dgx38h2x3nx8ye2s075e6apl0": {210, 30},
	"haqq1qp348cwad5sv2lg8j8nt3ptmmghxczcjueaggn": {151, 6},
	"haqq1qx98ggz3r0hnmllckzy5vqe3u76c06rurxqhmr": {179, 4},
	"haqq1qgta9kd3426et77c6pflf5y4h0ksa43zynhr2r": {312, 15},
	"haqq1q2w275mjjrjv7tkn98rhjzv6ldt6qj88z40lls": {2211, 1},
	"haqq1q0agznk82xnxkuwuxyl6vt8pxen50znzh7j2qg": {950, 0},
	"haqq1qn3nvnnet5kphneq0ckvnyuq6ferhlq67fz6lz": {2772, 6},
	"haqq1q42z9lxl9f3hyp6j7amq9txuh34kdxtk92ay5l": {327, 0},
	"haqq1qhxmcn2x2g4l4eyfusurtg77smcyz975qu0jk6": {180, 4},
	"haqq1pq7jp3k0ngz5ghp7s7lep59v4z76u3zl4a6mhj": {256, 0},
	"haqq1pzzv24t9g7jps5qgu4cc4w486c852x64d2shmz": {1611, 7},
	"haqq1pxzc8k9mxl4kf9fc4msn5446usv4p2n9lsw4af": {23, 0},
	"haqq1p8k6xk94u24vv9dmxu3vkgg43fs3v72g05dun6": {3146, 28},
	"haqq1p2vhterfrg02rlcw6x9hxn5qmpl0mg65hn5uuh": {181, 5},
	"haqq1psatth54608v0p6c2axdupwga6rlgy833x6gc5": {353, 0},
	"haqq1p64cacmcwwe0fvycm55e98yfqya0hkhv7ps4jn": {978713, 0},
	"haqq1pawc0xyh5f6ct52ajk8njru725nrks680t48ny": {271, 4},
	"haqq1zp2hep5w80jjm9tnpuk7aqxmg0qhl88xkjmjdl": {283, 1},
	"haqq1z2fjz5d6vs25xmhpeuwt23st25utdl4q6xy9jj": {1827772, 2},
	"haqq1zdd9unxe5hhz652542t0q5vq86t8mwu0yqnvrn": {64, 0},
	"haqq1z5sstr9xw9kyp0vk7jxntcgt49m3qnhkaqpzx5": {1129, 0},
	"haqq1rrujfp4wdux5e9sqpkmqfq858q8625uyazs7t3": {1546, 13},
	"haqq1rytep72msym6e5eyeh4sjm295uf0k7z8hk2y54": {1007, 0},
	"haqq1r9xv0eeq27hckqugtt3mfvwfqp7r6lmlz83gas": {70238, 1},
	"haqq1r8mmpzexzj8yt3y0aqls2c5ytdm65wuyjl2ms3": {282, 0},
	"haqq1rvcrc9cdq465xq38j6r8f0aarezg6v03jrtw4w": {1113316, 1},
	"haqq1rjev4p57pda04rpt34qf52trrtqszedg63qy5x": {3476, 0},
	"haqq1r526j8uf9avjqkzlg4m42sj8vcmsu4leejzc5w": {47, 13},
	"haqq1rh249lzjptgajta44qv6zugmq8v0xtu344snul": {253, 0},
	"haqq1ra55j7a7k8d7xx0z28mvetstnf2g69cjwtky5j": {398, 1},
	"haqq1yq6ckya0y5cuwa34cfkzjk57cuxkynx9mmlf94": {182, 0},
	"haqq1ygf3s8ljswkegr3u0gvawv5gyprh9qwpymw74n": {978724, 0},
	"haqq1ytud4sly5736g6g0y25ds60a69yl8mh9wl7p36": {169, 1},
	"haqq1yj78e9p73aet780rwzjsr33qr56es0tyrnz38p": {90, 22},
	"haqq1y46a9tl4a0y2n05fmynrdwwpr0gsetehu73dnu": {6830, 0},
	"haqq1y6qtj0zwz7f24eu3ywqgcqz8uuxyxx55gj6azu": {944970, 11},
	"haqq1ym9x4tp09w54zjsk8etx4gh6g080rvq2cqnux9": {211, 0},
	"haqq1yav0e5mqdkhc5mslcuju66qav2cfzp4jksktux": {99, 276},
	"haqq19q99p0vfp0hlns7v3ufc9urudnyfqp26nvpffu": {243, 21},
	"haqq19p8zc8ype7gkxm20kqtfd6gu7uyjqxejz0cst6": {949, 0},
	"haqq19vxt35gxnq7qnrtfatcdxajgnt70uly7nat7p4": {135, 7},
	"haqq1908t4ak9tt8h98937y5arwr9y66mtfngh0ymts": {70521, 1},
	"haqq19l2g0eas2th6xnhzlvjk9c078meelzjzvtvcrf": {1113335, 1},
	"haqq1xqnjk3ftdw798c8skxu6c9k0qm8n00fupmd0uc": {1828998, 0},
	"haqq1xzmucuwvpwh6cjhmcl7vha479dp5hvvz6xsg5s": {25, 59},
	"haqq1x23ghpnxkfevy2mcafxhtp3j9gutshsz2l4ez5": {950378, 1},
	"haqq1xvzm5zha02afv9sae8gnuhlxenn6383z0qk43a": {1113525, 1},
	"haqq1xvtkr46rgmdw60u4c58g72n8lq24vfkyh6clty": {71, 0},
	"haqq1x4v95wmzfquswp62026ekduuavms79yp6sl0ch": {331, 35},
	"haqq1xutax8wxt758h629npyduylkv22arvqa9tty5q": {335, 0},
	"haqq1xl0ruu6dqeuvpe24nh0v5zn9pgjd48lre0cdgx": {1113389, 3},
	"haqq18q2p9vlyl858n08rrp52vmjq743u02uzdupm3a": {1113327, 1},
	"haqq18z08hgj8ys5krraeucqtvsrlggjr4cx9qgkpqw": {1114318, 0},
	"haqq188g2j6xepwzt9zlvesxxj9ke8pfr8hx5s376yz": {27, 0},
	"haqq188e22t9x6k5963cn68gzan3yrp62slx6hftf7k": {69282, 1},
	"haqq18wf0d2dfjeldn4gxu36ptj8y65h7k2k26hkqvg": {3563, 0},
	"haqq185vnf4u3ldes0lj407nz83kjyuth0jcq5ymuqw": {1743522, 1},
	"haqq18hvxczazxlmcn06hw0h25d75frn9xsyshqjcrr": {420, 0},
	"haqq18etnw896uk6au2tp6q2zungt6su970pwkgu028": {393, 3},
	"haqq18a7nzz4s86drk8jnsz6v8mtpkmqv83a5tmqhry": {1131372, 0},
	"haqq1gpz0rexmnp42z2wargdughz0uwnqdnd6luy2c2": {2854, 0},
	"haqq1g85t7pyqe2rk4mmpm2nxgjdp5cp73t2hcxk3dl": {289, 64},
	"haqq1g8e2dw6vfhtu7s4lr4rfwh60f4zskqe73f27ne": {262, 0},
	"haqq1g2ku63ljhtpgxngcl75tsn9780n92hk7ew8lz8": {119, 2},
	"haqq1gdrw2ng0r6s9tlypd8v4chrv92w0umvzvexdw6": {219, 14},
	"haqq1g03ue3nnzsrdz73p0ttjg0r83z25vgy80n3fx7": {1324, 0},
	"haqq1g054py69kndwjmmahjudyaucz6tcshzukdvsrz": {534544, 5},
	"haqq1g3ynm077hrtpcyzjn4nyu2v4z4zmaz9eaqgk8s": {87, 0},
	"haqq1g3krfgyyj2hnt4rzzey00w5z6r5lzpzgkp0jlm": {1114244, 1},
	"haqq1gcy6l7tsfce7009smc5yacz4k6mdzr43p7hk6g": {212, 0},
	"haqq1gcmt3yqq92vytgq0sceddeujfgf459p6vzp2pv": {91, 0},
	"haqq1g6wkcczc9vq0k87g6a0tx0mwnu6vxfgfd0aqv4": {171, 0},
	"haqq1gm84u643vwkujjnwg0rfkm4463xp9sq6ll5yp4": {1289161, 2},
	"haqq1fx5ngwyxhpa87875wzaqqe9sylvr7qykslxf0n": {1113319, 1},
	"haqq1fgwaqyrfwyumzc97h2d27x2gant5lf8pn6ffdh": {1673, 24},
	"haqq1f3442mrprlqlhwze30pg5xpe4th0nggcgxux68": {1209, 12},
	"haqq1f773y9w88sef3wl3p6gf6mmsus3347q3syr5fs": {1210, 0},
	"haqq12dy2yvfaptg2dudghp4y556y04hsl557tq79zq": {661112, 1},
	"haqq125ff4kkaett4nld7ju06tdgew4cu3x9j7mgfzh": {1297312, 2},
	"haqq12lfcj8s2p200wtwm4r97g5jlhtke2qtzvfuehy": {356, 0},
	"haqq1trxvsfttdnzzazqfqu4wwu4fajld97lex5xxyy": {2290, 7},
	"haqq1tvn5kd6rqmjsvmvm2aypgxec9pvkq449kckwty": {1113523, 1},
	"haqq1twqjy4slv7s9ate78fajrhgu5ckup7z39rt3de": {69882, 1},
	"haqq1t5hcde8z5uq30s45hy2kazfpz05wvtz8nzsavj": {255, 0},
	"haqq1vzxkmfpyv853yc9qg8skthl2ypuzn7jprxjkzc": {1113388, 3},
	"haqq1vyv5mkrwkkydaj2gdh4w4pkdappxw5up2turq4": {160, 14},
	"haqq1vgyh727vg6ghxhmeuymz46t6w0z73fs2d4u5lc": {147, 0},
	"haqq1vf4g0xf8sngmmzta7wrraqd9qlafqfjgsgtced": {68, 1},
	"haqq1vdl2m2vjhtpgqqcgsr3j9w5crywtzjnzpzs6qm": {714433, 0},
	"haqq1vsrywh036tshafzp03ad8zdsueprfss45h0sv6": {740, 1855},
	"haqq1dpypafr8rj7snzgm936z0500anjzzrzxkmr8um": {26, 29},
	"haqq1drz5sqsv40u6r37lldt7manscrmnlm45mggwc0": {23181, 2},
	"haqq1d83fz57tysqdyfhu0sua7dzy29y4h7dqe30qer": {3520, 0},
	"haqq1dvx84uamrpzxzfmc5teflvt3z0kjaqg8ehz63k": {270, 0},
	"haqq1d3xrxcf98zrpxf93e6q9m5u4hcn4a7d8qsw44l": {137, 0},
	"haqq1dnvfqv563r9sre608uk5flx3x5wm30jk04jel7": {978708, 0},
	"haqq1dn3gzlp55mjkeucqdx4e43uk5nr3us7m5hf3cw": {1759, 19},
	"haqq1dkjfccqdr57ua78zvrx898esck7r55acsy62tu": {635936, 18},
	"haqq1da02g7tezx2xkc997t9nsz5lgc8smk7km5y2hk": {70604, 2},
	"haqq1wpupww426asz2pek824z8sc5gufez9hw6a3nwu": {1289166, 0},
	"haqq1wffxqaajn4qqqjkut64ez5faz8tevkvpu2senl": {21225, 0},
	"haqq1wwsssy28yl4rg464tzs9cfnefjhlvkhdj087sf": {254, 0},
	"haqq1w5lstjcpzgmusxxsnc9r87mcmlj8r2vujvgla8": {213, 1},
	"haqq1w4rjmxwdcz8rwyga02u5p64wutcp6gwzp3w48h": {1114262, 0},
	"haqq1wufxyqew7984n6y95g7tz7wwfus7guqv7qu6nt": {801227, 1},
	"haqq10qrnu0erlhd9p2yqah6cnc3wt7r8mznryzrwxy": {104, 39},
	"haqq10pxfedw2x2p0cauzwjvcujc5nx4wp2dun7qusk": {31670, 3},
	"haqq10zzcjxfjyhswfel34zy30fu92uac3etahrctas": {1102, 8},
	"haqq108rgs98c2t833he0gepqrfufnwc0vytpg4nnwe": {468812, 0},
	"haqq10wea29z56k0qmc3vdw5djcktpjylx0n3rmd9n7": {801244, 1},
	"haqq10sgvvm9xe4h9t4pwq7l496s9va6xxyup62w6v6": {162, 7},
	"haqq1058qjmrueq55cmlngscn2ew3xcl43vfknlun3c": {370, 47},
	"haqq1sqearnv7kghkf75hsta43zgm7p7huux4ee4ph8": {281, 35},
	"haqq1sxuahz53knmqzhvnyhv824dp3syc4sx370ta82": {345, 0},
	"haqq1s0u9vstlkg4h8n4x7cejz38xtpl6kcypl5he97": {73, 0},
	"haqq1s5ftly77vj62s3jexjcwn6wn6ua6zztszewhg4": {1113308, 1},
	"haqq1shtwlvrqla423gdyefpy35pj75vcdafw3vaham": {70777, 1},
	"haqq1sc389ce90s7n2wrzp9dlsqwvnxkkzl9ax83zya": {1670, 0},
	"haqq1s7q4lw2gyuy0hvmdga3h0jyps6ldu4kwx4a7w3": {437, 18},
	"haqq13q6750weljcx4tewqdd3cutv07wep9atklqsac": {172, 0},
	"haqq13dveqk34exxq2hp42lf4z6jte555ufpjxfcq5t": {175, 0},
	"haqq134lgt922z8vmm5mg7yjdegdrdlsythl3umq5mu": {72, 0},
	"haqq13mhwtzn22qtsumle090leuq9mn2efurmrqe54q": {2549, 0},
	"haqq13uv3xszdu0aqug8rq5y7y5fpnus44hqrksnw5j": {1113343, 3},
	"haqq1jy4rhr8kqr9a6u0lcs3yaqzszgs5sg6x38xqds": {17445, 54},
	"haqq1jw337xvz5z572xpdd2h88taepnfq4t9rwpc624": {1792, 0},
	"haqq1jntputal4wsf332qt3fuv0gr8j794ak9cgun8e": {1057, 14},
	"haqq1jnk82c4zcwyfyajgt2yddcyz0hsyfkwnz4tt6m": {1104, 0},
	"haqq1jh375g33t6l3kd5wjhmscju2kyfezfkjyj5n4p": {3576, 0},
	"haqq1nqnxp6sf5nsq7239sp02da4slzgx4jaq7ly3dy": {292354, 0},
	"haqq1nwk6lajqnup9kc77mfp89fjejegn5s0hsjx5s7": {70429, 1},
	"haqq1nssne255e94mh63ntpnkgvn2qjr7kzvrq50u0v": {338, 13},
	"haqq1n38n065wwaxshrf3zxw4apdaqv9u3yhumzgyj3": {77, 21},
	"haqq1n4ct56gyzergcufjcffwxhfz3x2hnu95cqqz9v": {92, 8},
	"haqq1nhj7dffdyut62vkdrs7e20s3fqsgzlx6ly52yl": {74, 15},
	"haqq15xh3kgzkyfha4lmmcjtyvfm6m3qepl58zurse5": {635482, 2},
	"haqq152wt6su6h099knjdfq7eq0w2rvxy8s79yzfh0x": {1114257, 0},
	"haqq15vkeay725cch4g582z7lmse3tlc3m54nwdvmcn": {214, 0},
	"haqq15dnugu0lakanyvre8c927pzu8rjhahucvkp7wx": {70, 36},
	"haqq15sqkc94hvdlam5kr6nwutjeyeepmfr0xceshz4": {3521, 0},
	"haqq15uy0r690cq6y4yufkcs987c5k3h2gsjtq8kjtl": {31465, 3},
	"haqq1497ds93u23varcq32gg635c6tkvslqllgxfy5q": {1743503, 1},
	"haqq14268an9pn60lq80qmv9e3wque7c95dpkmjsnha": {1629, 1},
	"haqq14t43ptdqsvevznqf5knf9g94p88nhr0zhz386w": {78, 1},
	"haqq14d9h3xddz2jvx6zqnxup8272rgh5x4dlkxpa9h": {1114321, 0},
	"haqq14s98edrkmucdkenldar5tgzxp9nx02wfqcez84": {382, 0},
	"haqq14h03vayvjyc8c2k2dnjgtn3jn2zjvjhgg3jqqx": {75, 0},
	"haqq1kq7vc3euw3r3uyhrzj4cs9ccnm4a44akl6uu97": {70345, 1},
	"haqq1k9qe4tk99y6sq0yf77aev3ahuhc7yaeyqutwft": {2367, 0},
	"haqq1kv30peffjfau2du06nm2v3yzjdyl5qh30a0gsg": {133, 0},
	"haqq1k5tkey8ym6596jsmsfkxpfhswuhv9xk5q5n362": {675, 5},
	"haqq1kmfwuaha4dj5y5us46ga99t69vs3pf9de2t73k": {70090, 1},
	"haqq1kawnyp8w7ydk9fgtvp7m8t8kqf0vypykr3rj7v": {1743530, 1},
	"haqq1kl3rag8qgs8zlfj6lat9ltvwz9ar3ujevnedaj": {10540, 5},
	"haqq1hjcc2e9hhjcx7g7lhh7d37eafcazce98z3cusy": {239, 11},
	"haqq1h5hps8vm7ss9v599rxqrcuherzmwkgpuczewxc": {5438, 0},
	"haqq1hc04df4sh90vg6nw7urlg6p5ek7qqvrq2kc2hr": {32, 48},
	"haqq1hel3luwplhhpallueqenqz3zrfra9u55xh8a60": {834110, 3},
	"haqq1hurugc4tkpcgg2r5jjhnhvhppuqq9cnuaghhw6": {154, 1},
	"haqq1cqn4lxesxkyraktv5z6x7yl4ze48rz5ngjmgk3": {1113524, 1},
	"haqq1c8nsdx260ymrszv3gul52uvnj4p0l7jkpe7rcl": {29, 45},
	"haqq1c2cw25t8ve4xfnfwtt9rqzerclux4pdc67mu34": {1106, 6},
	"haqq1eyenhtzxs654dzl09dh3kmxyl38p225uy9v52c": {76, 1},
	"haqq1eshrk5lvw3q2ek5905dktdgrj3sfg9xflhuz8c": {190, 0},
	"haqq1enpjs3wxcp093zg8c0k0y63pnlfnnck5skd2nz": {3145, 1},
	"haqq1e5x2jv0q8k8y5cw5lxvez7rque2g4s6m7zrl9d": {1113526, 1},
	"haqq1e5x7xrfs72y3axlpn7cyypresax3yhr505uzhf": {946, 54},
	"haqq1eld02pp33sgj0xxjqgjwws6ddyn03v2m5m2f76": {1113527, 1},
	"haqq16ylhc5yq59h4wn6qflwsmk9232rqclt84azuv0": {993, 0},
	"haqq16xfdgsz8kydurul9jm3up3y3n8dd4hhqqz6wrq": {1103, 0},
	"haqq16s7jp8g2s6x2n4ls2l6atkzj67lmgdzayk338j": {69745, 1},
	"haqq16jqnt2tqy7gr6le9yx5t27e8fkk0swsxa37pn9": {2170, 0},
	"haqq165gc5vg9gyspmvrnz5y8y58nvqcrj7ee84l95x": {1134349, 0},
	"haqq164q7t469qw6v8knzcydt333ps447lqgzmk8tvr": {1827705, 1},
	"haqq1640x94ca3ac4jelyr0c080y8m380wfpn9gdvv5": {177, 4},
	"haqq16k82fxutdsyk9m859cx4uuswt5jg5e689n9gx8": {48, 63},
	"haqq16alfvfeluldkwx34hweetn0xuk4rzz353629hp": {100, 139},
	"haqq1mxcaug6cj8lyhvkwkz46ef7euzas27gq4mc3qy": {967208, 0},
	"haqq1mvtp7c4xh5haf84hy8gdfec9jcq7nc2gj8c94j": {1113331, 1},
	"haqq1m3eej28e0lu8ac0f924u8kev4huf92t3a5hu00": {173, 0},
	"haqq1mh4me4475ezn90fkfpvn6cqz9xswgudumz6g8e": {183, 19},
	"haqq1m629fqt2qf8y3af65ns5zfy49z36ztuyc8j6h9": {3258, 9},
	"haqq1upytyk7nm0d9qj7jfjslakanmqlvhqf9tugvtn": {46569, 0},
	"haqq1uzleh48vrx26z5mpxdjzzxfp3gv3wwlfzvdkhn": {1743496, 1},
	"haqq1uf5zujcrzpukztj2jypu8z9gtqyu4e679ktqrp": {643, 0},
	"haqq1ufur5wjg8mx7mzhtehwaaawthwhmeke0c5y5jz": {1141608, 1},
	"haqq1ujx832jccw9e5yea6mpd752zujfg48fefcwt8m": {459, 0},
	"haqq1unl4ynas357da4s5h7eycancssehs0nx26aymp": {184, 13},
	"haqq1u5pmps96mzs8qxegc6kl92eylqv8kjetndvczn": {85, 7},
	"haqq1ue838w7cq878pf5epsxfswuhu83fa4qe7t4qgz": {358, 17},
	"haqq1uekkrggz4gk6x89ymezm87tje3a876nuc62ydv": {1113345, 6},
	"haqq1um06w9d5z4kaqshec2v4yec6m6nfrt965xmytu": {2200, 0},
	"haqq1uus2dz0uy0ye26zphtrv47gq7n86zq7fka475l": {978721, 0},
	"haqq1afa2p6fec4jv2cp6teyu2q4jw77kf0r37la4uf": {174, 0},
	"haqq1amh5mr4vrp6tyxwm9q3s5hpjw0tvhghgu3l7s7": {640, 0},
	"haqq17q26vcrcc6nkggrruq6n7304y3pgptd0zu0k70": {275, 14},
	"haqq17y7y20f93n4k86pnpuufrnd02mevg4fyq4cvnn": {1114236, 0},
	"haqq17lcp28a3cgt5pj5z4ecpvph8h356c00tungtny": {427, 0},
	"haqq1lp5753cvk2vh0kt2chdchgh96h6lcqrzhtm88z": {24, 0},
	"haqq1lxwsfesv32qvycekeda6dqffxfq252l36cx4xe": {313, 15},
	"haqq1lvp4nuns5y6t5pvrttfaq6aecern5rxc87875x": {530026, 3},
	"haqq1lssuvksg8ehg0qmqnqth7y4t2ueysw0wlslkly": {44, 19},
	"haqq1l39jvq7t28h45wkps3uplld85j4jd904greylc": {1178940, 2},
	"haqq1l36rjxda4fr7epzlna5zxzruyknge9ltl43tza": {508, 1},
	"haqq1lld8gzg7urma5xlxqd27xmpug9cn5svz94a5f7": {384, 0},
}

func restoreAccounts(ctx sdk.Context, ak authkeeper.AccountKeeper) error {
	logger := ctx.Logger()
	logger.Info("Restoring accounts data...")

	keys := make([]string, 0, len(accounts))
	for k := range accounts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		accAddr := sdk.MustAccAddressFromBech32(k)
		acc := ak.GetAccount(ctx, accAddr)
		if acc == nil {
			return fmt.Errorf("account not found: %s", k)
		}

		oldNum := acc.GetAccountNumber()
		if err := acc.SetAccountNumber(accounts[k].accNum); err != nil {
			return fmt.Errorf("failed to restore account number: %s: %d", k, accounts[k].accNum)
		}

		oldSeq := acc.GetSequence()
		newSeq := oldSeq + accounts[k].seq
		if err := acc.SetSequence(accounts[k].seq + acc.GetSequence()); err != nil {
			return fmt.Errorf("failed to restore sequence: %s: %d", k, newSeq)
		}

		ak.SetAccount(ctx, acc)
		logger.Info(fmt.Sprintf("Restored account - %s: num %d -> %d; seq %d -> %d", k, oldNum, accounts[k].accNum, oldSeq, newSeq))
	}

	return nil
}
