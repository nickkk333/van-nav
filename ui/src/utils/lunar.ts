// 紧凑农历工具：1900-2100 年农历数据 + 节气/节日查询
// 数据来源为公开农历编码（ roliu/lunar 参考算法的简化实现）
const INFO = [0x04bd8,0x04ae0,0x0a570,0x054d5,0x0d260,0x0d950,0x16554,0x056a0,0x09ad0,0x055d2,0x04ae0,0x0a5b6,0x0a4d0,0x0d250,0x1d255,0x0b540,0x0d6a0,0x0ada2,0x095b0,0x14977,0x04970,0x0a4b0,0x0b4b5,0x06a50,0x06d40,0x1ab54,0x02b60,0x09570,0x052f2,0x04970,0x06566,0x0d4a0,0x0ea50,0x06e95,0x05ad0,0x02b60,0x186e3,0x092e0,0x1c8d7,0x0c950,0x0d4a0,0x1d8a6,0x0b550,0x056a0,0x1a5b4,0x025d0,0x092d0,0x0d2b2,0x0a950,0x0b557,0x06ca0,0x0b550,0x15355,0x04da0,0x0a5b0,0x14573,0x052b0,0x0a9a8,0x0e950,0x06aa0,0x0aea6,0x0ab50,0x04b60,0x0aae4,0x0a570,0x05260,0x0f263,0x0d950,0x05b57,0x056a0,0x096d0,0x04dd5,0x04ad0,0x0a4d0,0x0d4d4,0x0d250,0x0d558,0x0b540,0x0b6a0,0x195a6,0x095b0,0x049b0,0x0a974,0x0a4b0,0x0b27a,0x06a50,0x06d40,0x0af46,0x0ab60,0x09570,0x04af5,0x04970,0x064b0,0x074a3,0x0ea50,0x06b58,0x055c0,0x0ab60,0x096d5,0x092e0,0x0c960,0x0d954,0x0d4a0,0x0da50,0x07552,0x056a0,0x0abb7,0x025d0,0x092d0,0x0cab5,0x0a950,0x0b4a0,0x0baa4,0x0ad50,0x055d9,0x04ba0,0x0a5b0,0x15176,0x052b0,0x0a930,0x07954,0x06aa0,0x0ad50,0x05b52,0x04b60,0x0a6e6,0x0a4e0,0x0d260,0x0ea65,0x0d530,0x05aa0,0x076a3,0x096d0,0x04afb,0x04ad0,0x0a4d0,0x1d0b6,0x0d250,0x0d520,0x0dd45,0x0b5a0,0x056d0,0x055b2,0x049b0,0x0a577,0x0a4b0,0x0aa50,0x1b255,0x06d20,0x0ada0,0x14b63,0x09370,0x049f8,0x04970,0x064b0,0x168a6,0x0ea50,0x06b20,0x1a6c4,0x0aae0,0x0a2e0,0x0d2e3,0x0c960,0x0d557,0x0d4a0,0x0da50,0x05d55,0x056a0,0x0a6d0,0x055d4,0x052d0,0x0a9b8,0x0a950,0x0b4a0,0x0b6a6,0x0ad50,0x055a0,0x0aba4,0x0a5b0,0x052b0,0x0b273,0x06930,0x07337,0x06aa0,0x0ad50,0x14b55,0x04b60,0x0a570,0x054e4,0x0d160,0x0e968,0x0d520,0x0daa0,0x16aa6,0x056d0,0x04ae0,0x0a9d4,0x0a2d0,0x0d150,0x0f252,0x0d520,]
const leapMonth = (y: number) => INFO[y - 1900] & 0xf
const leapDays = (y: number) => leapMonth(y) ? ((INFO[y - 1900] & 0x10000) ? 30 : 29) : 0
const monthDays = (y: number, m: number) => (INFO[y - 1900] & (0x10000 >> m)) ? 30 : 29
const yearDays = (y: number) => { let s = 348; for (let i = 0x8000; i > 0x8; i >>= 1) s += (INFO[y - 1900] & i) ? 1 : 0; return s + leapDays(y) }
const NONG = ['初一','初二','初三','初四','初五','初六','初七','初八','初九','初十','十一','十二','十三','十四','十五','十六','十七','十八','十九','二十','廿一','廿二','廿三','廿四','廿五','廿六','廿七','廿八','廿九','三十']
const YUE = ['正','二','三','四','五','六','七','八','九','十','冬','腊']
export interface LunarInfo { month: string; day: string }
export const solarToLunar = (y: number, m: number, d: number): LunarInfo => {
  let offset = Math.floor((Date.UTC(y, m - 1, d) - Date.UTC(1900, 0, 31)) / 86400000)
  let year = 1900
  while (year < 2101 && offset > 0) { const d = yearDays(year); if (offset < d) break; offset -= d; year++ }
  if (year > 2100) return { month: '', day: '' }
  const leap = leapMonth(year)
  let isLeap = false, month = 1
  while (month <= 12 && offset > 0) {
    let dm = monthDays(year, month)
    if (month === leap + 1 && !isLeap) { isLeap = true; dm = leapDays(year); month--; }
    else isLeap = false
    if (offset < dm) break
    offset -= dm; month++
  }
  if (month > 12) return { month: '', day: '' }
  const mStr = (isLeap ? '闰' : '') + YUE[month - 1] + '月'
  // day 恒为农历日名（offset===0 即初一），单元格里「初一显示成月份」的约定由调用方处理
  return { month: mStr, day: NONG[offset] }
}
// 节气：按固定日期近似（误差±1 天，仅用于日历角标展示；精确节气需查表计算，这里不引入）
const JIEQI_DAY: Record<string, [string, string]> = {
  '1': ['小寒', '大寒'], '2': ['立春', '雨水'], '3': ['惊蛰', '春分'], '4': ['清明', '谷雨'],
  '5': ['立夏', '小满'], '6': ['芒种', '夏至'], '7': ['小暑', '大暑'], '8': ['立秋', '处暑'],
  '9': ['白露', '秋分'], '10': ['寒露', '霜降'], '11': ['立冬', '小雪'], '12': ['大雪', '冬至'],
}
const JIEQI_DATE: Record<string, [number, number]> = {
  '1': [6, 20], '2': [4, 19], '3': [6, 21], '4': [5, 20], '5': [6, 21], '6': [6, 22],
  '7': [7, 23], '8': [8, 23], '9': [8, 23], '10': [8, 24], '11': [8, 22], '12': [7, 22],
}
export const getJieQi = (y: number, m: number, d: number): string => {
  void y
  const [a, b] = JIEQI_DATE[String(m)]
  const [na, nb] = JIEQI_DAY[String(m)]
  if (d === a) return na
  if (d === b) return nb
  return ''
}
export const LUNAR_FEST = { '正月初一': '春节', '正月十五': '元宵', '五月初五': '端午', '七月初七': '七夕', '八月十五': '中秋', '九月初九': '重阳', '腊月初八': '腊八', '腊月廿三': '小年' } as Record<string, string>
export const lunarFestival = (y: number, m: number, d: number): string => {
  const l = solarToLunar(y, m, d)
  return LUNAR_FEST[`${l.month.replace('闰', '')}${l.day}`] || ''
}
