// ─── 台灣縣市 + 鄉鎮市區二階選單資料（含主要市區郵遞區號） ──

export interface DistrictInfo {
  name: string;
  zip: string;
}

export interface RegionInfo {
  city: string;
  districts: DistrictInfo[];
}

export const TAIWAN_REGIONS: RegionInfo[] = [
  {
    city: "台北市",
    districts: [
      { name: "中正區", zip: "100" },
      { name: "大同區", zip: "103" },
      { name: "中山區", zip: "104" },
      { name: "松山區", zip: "105" },
      { name: "大安區", zip: "106" },
      { name: "萬華區", zip: "108" },
      { name: "信義區", zip: "110" },
      { name: "士林區", zip: "111" },
      { name: "北投區", zip: "112" },
      { name: "內湖區", zip: "114" },
      { name: "南港區", zip: "115" },
      { name: "文山區", zip: "116" },
    ],
  },
  {
    city: "新北市",
    districts: [
      { name: "板橋區", zip: "220" },
      { name: "三重區", zip: "241" },
      { name: "中和區", zip: "235" },
      { name: "永和區", zip: "234" },
      { name: "新莊區", zip: "242" },
      { name: "新店區", zip: "231" },
      { name: "土城區", zip: "236" },
      { name: "蘆洲區", zip: "247" },
      { name: "樹林區", zip: "248" },
      { name: "汐止區", zip: "221" },
      { name: "淡水區", zip: "251" },
      { name: "鶯歌區", zip: "239" },
    ],
  },
  {
    city: "基隆市",
    districts: [
      { name: "仁愛區", zip: "200" },
      { name: "信義區", zip: "201" },
      { name: "中正區", zip: "202" },
      { name: "安樂區", zip: "203" },
      { name: "暖暖區", zip: "205" },
      { name: "七堵區", zip: "206" },
    ],
  },
  {
    city: "桃園市",
    districts: [
      { name: "桃園區", zip: "330" },
      { name: "中壢區", zip: "320" },
      { name: "平鎮區", zip: "324" },
      { name: "八德區", zip: "334" },
      { name: "楊梅區", zip: "326" },
      { name: "蘆竹區", zip: "338" },
      { name: "龜山區", zip: "333" },
      { name: "大園區", zip: "337" },
    ],
  },
  {
    city: "新竹市",
    districts: [
      { name: "東區", zip: "300" },
      { name: "北區", zip: "300" },
      { name: "香山區", zip: "300" },
    ],
  },
  {
    city: "新竹縣",
    districts: [
      { name: "竹北市", zip: "302" },
      { name: "竹東鎮", zip: "310" },
      { name: "新豐鄉", zip: "304" },
      { name: "湖口鄉", zip: "303" },
      { name: "關西鎮", zip: "306" },
    ],
  },
  {
    city: "苗栗縣",
    districts: [
      { name: "苗栗市", zip: "360" },
      { name: "頭份市", zip: "351" },
      { name: "竹南鎮", zip: "350" },
      { name: "通霄鎮", zip: "357" },
      { name: "苑裡鎮", zip: "358" },
    ],
  },
  {
    city: "台中市",
    districts: [
      { name: "中區", zip: "400" },
      { name: "東區", zip: "401" },
      { name: "南區", zip: "402" },
      { name: "西區", zip: "403" },
      { name: "北區", zip: "404" },
      { name: "西屯區", zip: "407" },
      { name: "南屯區", zip: "408" },
      { name: "北屯區", zip: "406" },
      { name: "豐原區", zip: "420" },
      { name: "大里區", zip: "412" },
      { name: "太平區", zip: "411" },
      { name: "沙鹿區", zip: "433" },
    ],
  },
  {
    city: "彰化縣",
    districts: [
      { name: "彰化市", zip: "500" },
      { name: "員林市", zip: "510" },
      { name: "鹿港鎮", zip: "505" },
      { name: "和美鎮", zip: "508" },
      { name: "北斗鎮", zip: "524" },
    ],
  },
  {
    city: "南投縣",
    districts: [
      { name: "南投市", zip: "540" },
      { name: "草屯鎮", zip: "542" },
      { name: "埔里鎮", zip: "545" },
      { name: "竹山鎮", zip: "541" },
    ],
  },
  {
    city: "嘉義市",
    districts: [
      { name: "東區", zip: "600" },
      { name: "西區", zip: "600" },
    ],
  },
  {
    city: "嘉義縣",
    districts: [
      { name: "朴子市", zip: "613" },
      { name: "太保市", zip: "612" },
      { name: "民雄鄉", zip: "621" },
      { name: "大林鎮", zip: "604" },
    ],
  },
  {
    city: "台南市",
    districts: [
      { name: "中西區", zip: "700" },
      { name: "東區", zip: "701" },
      { name: "南區", zip: "702" },
      { name: "北區", zip: "704" },
      { name: "安平區", zip: "708" },
      { name: "安南區", zip: "709" },
      { name: "永康區", zip: "710" },
      { name: "新營區", zip: "730" },
      { name: "仁德區", zip: "717" },
    ],
  },
  {
    city: "高雄市",
    districts: [
      { name: "新興區", zip: "800" },
      { name: "前金區", zip: "801" },
      { name: "苓雅區", zip: "802" },
      { name: "鹽埕區", zip: "803" },
      { name: "鼓山區", zip: "804" },
      { name: "左營區", zip: "813" },
      { name: "三民區", zip: "807" },
      { name: "楠梓區", zip: "811" },
      { name: "前鎮區", zip: "806" },
      { name: "鳳山區", zip: "830" },
    ],
  },
  {
    city: "屏東縣",
    districts: [
      { name: "屏東市", zip: "900" },
      { name: "潮州鎮", zip: "920" },
      { name: "東港鎮", zip: "928" },
      { name: "恆春鎮", zip: "946" },
    ],
  },
  {
    city: "宜蘭縣",
    districts: [
      { name: "宜蘭市", zip: "260" },
      { name: "羅東鎮", zip: "265" },
      { name: "礁溪鄉", zip: "262" },
      { name: "冬山鄉", zip: "269" },
    ],
  },
  {
    city: "花蓮縣",
    districts: [
      { name: "花蓮市", zip: "970" },
      { name: "吉安鄉", zip: "973" },
      { name: "新城鄉", zip: "971" },
      { name: "玉里鎮", zip: "984" },
    ],
  },
  {
    city: "台東縣",
    districts: [
      { name: "台東市", zip: "950" },
      { name: "關山鎮", zip: "956" },
      { name: "成功鎮", zip: "961" },
    ],
  },
  {
    city: "澎湖縣",
    districts: [
      { name: "馬公市", zip: "880" },
      { name: "湖西鄉", zip: "885" },
    ],
  },
  {
    city: "金門縣",
    districts: [
      { name: "金城鎮", zip: "893" },
      { name: "金湖鎮", zip: "891" },
    ],
  },
  {
    city: "連江縣",
    districts: [
      { name: "南竿鄉", zip: "209" },
      { name: "北竿鄉", zip: "210" },
    ],
  },
];

/** 依縣市名取得行政區清單 */
export function getDistricts(city: string): DistrictInfo[] {
  return TAIWAN_REGIONS.find((r) => r.city === city)?.districts ?? [];
}
