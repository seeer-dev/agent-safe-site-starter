// ─── 超商取貨假門市資料（7-11 / 全家，供店到店選擇器） ──────

export interface CvsStore {
  id: string; // 門市店號
  brand: "7-11" | "FamilyMart";
  name: string;
  city: string;
  address: string;
}

/** 7-ELEVEN 門市（約 22 間） */
export const SEVEN_ELEVEN_STORES: CvsStore[] = [
  { id: "016239", brand: "7-11", name: "7-ELEVEN 台大門市", city: "台北市", address: "台北市大安區羅斯福路四段1號1樓" },
  { id: "018927", brand: "7-11", name: "7-ELEVEN 誠品門市", city: "台北市", address: "台北市信義區松高路11號B2" },
  { id: "013841", brand: "7-11", name: "7-ELEVEN 民生門市", city: "台北市", address: "台北市松山區民生東路四段56號" },
  { id: "011204", brand: "7-11", name: "7-ELEVEN 西門門市", city: "台北市", address: "台北市萬華區成都路52號" },
  { id: "019213", brand: "7-11", name: "7-ELEVEN 天母門市", city: "台北市", address: "台北市士林區中山北路六段245號" },
  { id: "023417", brand: "7-11", name: "7-ELEVEN 板橋車站門市", city: "新北市", address: "新北市板橋區縣民大道二段7號B1" },
  { id: "025180", brand: "7-11", name: "7-ELEVEN 新店中央門市", city: "新北市", address: "新北市新店區中央路155號1樓" },
  { id: "027293", brand: "7-11", name: "7-ELEVEN 淡水英專門市", city: "新北市", address: "新北市淡水區英專路69號" },
  { id: "028756", brand: "7-11", name: "7-ELEVEN 桃園中正門市", city: "桃園市", address: "桃園市桃園區中正路118號" },
  { id: "031284", brand: "7-11", name: "7-ELEVEN 中壢中央門市", city: "桃園市", address: "桃園市中壢區中央西路一段73號" },
  { id: "033925", brand: "7-11", name: "7-ELEVEN 青年門市", city: "桃園市", address: "桃園市八德區介壽路一段385號" },
  { id: "036472", brand: "7-11", name: "7-ELEVEN 新竹北大門市", city: "新竹市", address: "新竹市北區北大路256號" },
  { id: "038609", brand: "7-11", name: "7-ELEVEN 竹北光明門市", city: "新竹縣", address: "新竹縣竹北市光明一路210號" },
  { id: "041328", brand: "7-11", name: "7-ELEVEN 中科門市", city: "台中市", address: "台中市西屯區台灣大道四段1086號" },
  { id: "043571", brand: "7-11", name: "7-ELEVEN 勤美門市", city: "台中市", address: "台中市西區公益路68號地下一樓" },
  { id: "045106", brand: "7-11", name: "7-ELEVEN 逢甲福星門市", city: "台中市", address: "台中市西屯區福星路427號" },
  { id: "047852", brand: "7-11", name: "7-ELEVEN 豐原中正門市", city: "台中市", address: "台中市豐原區中正路127號" },
  { id: "052139", brand: "7-11", name: "7-ELEVEN 員林中山門市", city: "彰化縣", address: "彰化縣員林市中山路二段31號" },
  { id: "056378", brand: "7-11", name: "7-ELEVEN 嘉義中山門市", city: "嘉義市", address: "嘉義市東區中山路178號" },
  { id: "058462", brand: "7-11", name: "7-ELEVEN 台南南門門市", city: "台南市", address: "台南市中西區南門路113號" },
  { id: "061203", brand: "7-11", name: "7-ELEVEN 永康中華門市", city: "台南市", address: "台南市永康區中華路12號" },
  { id: "064817", brand: "7-11", name: "7-ELEVEN 高雄漢神門市", city: "高雄市", address: "高雄市前金區成功一路266號B1" },
  { id: "067349", brand: "7-11", name: "7-ELEVEN 左營高鐵門市", city: "高雄市", address: "高雄市左營區高鐵路105號B1" },
  { id: "069524", brand: "7-11", name: "7-ELEVEN 屏東民生門市", city: "屏東縣", address: "屏東縣屏東市民生路12-1號" },
  { id: "072361", brand: "7-11", name: "7-ELEVEN 宜蘭中山門市", city: "宜蘭縣", address: "宜蘭縣宜蘭市中山路三段126號" },
  { id: "075890", brand: "7-11", name: "7-ELEVEN 花蓮中正門市", city: "花蓮縣", address: "花蓮縣花蓮市中正路326號" },
];

/** 全家便利商店（約 22 間） */
export const FAMILYMART_STORES: CvsStore[] = [
  { id: "F281039", brand: "FamilyMart", name: "全家 台大新生店", city: "台北市", address: "台北市大安區新生南路三段76巷3號" },
  { id: "F287214", brand: "FamilyMart", name: "全家 信義威秀店", city: "台北市", address: "台北市信義區松壽路20號1樓" },
  { id: "F290467", brand: "FamilyMart", name: "全家 南京松江店", city: "台北市", address: "台北市中山區南京東路二段97號" },
  { id: "F293852", brand: "FamilyMart", name: "全家 士林文林店", city: "台北市", address: "台北市士林區文林路102號" },
  { id: "F296730", brand: "FamilyMart", name: "全家 萬華西寧店", city: "台北市", address: "台北市萬華區西寧南路72號" },
  { id: "F301246", brand: "FamilyMart", name: "全家 板橋文化店", city: "新北市", address: "新北市板橋區文化路一段25號" },
  { id: "F304871", brand: "FamilyMart", name: "全家 永和中山店", city: "新北市", address: "新北市永和區中山路一段283號" },
  { id: "F307935", brand: "FamilyMart", name: "全家 汐止建成店", city: "新北市", address: "新北市汐止區建成路56號" },
  { id: "F312408", brand: "FamilyMart", name: "全家 桃園中山店", city: "桃園市", address: "桃園市桃園區中山東路32號" },
  { id: "F315762", brand: "FamilyMart", name: "全家 中壢中央店", city: "桃園市", address: "桃園市中壢區中央西路二段51號" },
  { id: "F318924", brand: "FamilyMart", name: "全家 龜山文化店", city: "桃園市", address: "桃園市龜山區文化一路10巷3號" },
  { id: "F323517", brand: "FamilyMart", name: "全家 新竹光復店", city: "新竹市", address: "新竹市東區光復路一段276號" },
  { id: "F326843", brand: "FamilyMart", name: "全家 竹北莊敬店", city: "新竹縣", address: "新竹縣竹北市莊敬北路112號" },
  { id: "F331259", brand: "FamilyMart", name: "全家 台中公益店", city: "台中市", address: "台中市西區公益路161號" },
  { id: "F334670", brand: "FamilyMart", name: "全家 一中街店", city: "台中市", address: "台中市北區三民路三段120號" },
  { id: "F337182", brand: "FamilyMart", name: "全家 朝馬店", city: "台中市", address: "台中市西屯區台灣大道三段558號" },
  { id: "F342965", brand: "FamilyMart", name: "全家 豐原成功店", city: "台中市", address: "台中市豐原區成功路260號" },
  { id: "F347318", brand: "FamilyMart", name: "全家 彰化中正店", city: "彰化縣", address: "彰化縣彰化市中正路二段75號" },
  { id: "F351742", brand: "FamilyMart", name: "全家 嘉義民族店", city: "嘉義市", address: "嘉義市東區民族路146號" },
  { id: "F356280", brand: "FamilyMart", name: "全家 台南府前店", city: "台南市", address: "台南市中西區府前路二段140號" },
  { id: "F359417", brand: "FamilyMart", name: "全家 平鎮環南店", city: "桃園市", address: "桃園市平鎮區環南路三段41號" },
  { id: "F362853", brand: "FamilyMart", name: "全家 高雄五福店", city: "高雄市", address: "高雄市前金區五福三路63號" },
  { id: "F365109", brand: "FamilyMart", name: "全家 鳳山中正店", city: "高雄市", address: "高雄市鳳山區中正路118號" },
  { id: "F368472", brand: "FamilyMart", name: "全家 屏東廣東店", city: "屏東縣", address: "屏東縣屏東市廣東路552號" },
  { id: "F371638", brand: "FamilyMart", name: "全家 羅東公正店", city: "宜蘭縣", address: "宜蘭縣羅東鎮公正路155號" },
  { id: "F374951", brand: "FamilyMart", name: "全家 花蓮中山店", city: "花蓮縣", address: "花蓮縣花蓮市中山路142-1號" },
];

/** 依品牌取得門市清單 */
export function getStoresByBrand(brand: "7-11" | "FamilyMart"): CvsStore[] {
  return brand === "7-11" ? SEVEN_ELEVEN_STORES : FAMILYMART_STORES;
}
