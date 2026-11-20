import pkg from '../../package.json'

/**
 * 「关于应用」页面展示的信息。
 *
 * 收款码图片放在 ui/public/about/ 目录下（构建后会原样拷贝到站点根目录）：
 *   ui/public/about/wechat.png   ->  /about/wechat.png
 *   ui/public/about/alipay.png   ->  /about/alipay.png
 * 图片不存在时页面会显示占位框，换成自己的图片后无需改代码。
 */
export const ABOUT_INFO = {
  /** 站点标题为空时显示的应用名 */
  name: 'Van Nav',
  description: '专为 NAS 与自托管用户打造的轻量级、多用户私有云导航站。',
  developer: 'nickkk333',
  /** 开发者 GitHub 主页 / 项目仓库地址 */
  developerUrl: 'https://github.com/nickkk333/van-nav',
  /** 版本号直接取 ui/package.json，发版时只改这一处 */
  version: pkg.version,
  /** 版本号跳转的 GitHub Releases 页面 */
  releaseUrl: 'https://github.com/nickkk333/van-nav/releases',
  /** 微信 / 支付宝赞赏码（留空则显示占位框，隐藏对应项） */
  wechatQr: `${import.meta.env.BASE_URL}about/wechat.png`,
  alipayQr: `${import.meta.env.BASE_URL}about/alipay.png`,
}

export const ABOUT_SUPPORT_TIP = '如果您觉得这个项目对您有帮助，欢迎请作者喝杯咖啡'
