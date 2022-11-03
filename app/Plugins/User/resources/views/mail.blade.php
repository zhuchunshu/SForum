<div>
    <includetail>
        <div style="font:Verdana normal 14px;color:#000;">
            <div style="position:relative;">
                <div class="eml-w eml-w-sys-layout">
                    {{--页眉--}}
                    <div style="font-size: 0px;">
                        {{--分割线--}}
                        <div class="eml-w-sys-line">
                            <div class="eml-w-sys-line-left"></div>
                            <div class="eml-w-sys-line-right"></div>
                        </div>
                        {{--  LOGO --}}
                        <div class="eml-w-sys-logo">
                            <img src="https://rescdn.qqmail.com/node/wwqy/qymng/style/images/sass/independent/welcome_eml_logo.png" style="width: 34px; height: 24px;" onerror="">
                        </div>
                    </div>
                    {{--以下写正文--}}
                    <div class="eml-w-sys-content">
                        {{--以下写正文--}}
                        <div class="dragArea gen-group-list">

                            {{-- 普通的文本  --}}
                            <div class="gen-item">
                                <div class="eml-w-item-block" style="padding: 0px;">
                                    <div class="eml-w-phase-normal-16">你好，张三，你的注册验证码为：</div>
                                </div>
                            </div>

                            {{-- 普通的文本【自定义样式】  --}}
                            <div class="gen-item">
                                {{-- padding:行间距 --}}
                                <div class="eml-w-item-block" style="padding: 10px;">
                                    <div class="eml-w-phase-normal-16" style="color: red;text-align: center;font-weight:bold"> 123456 </div>
                                </div>
                            </div>

                            {{-- 普通的文本【自定义样式】  --}}
                            <div class="gen-item">
                                {{-- padding:行间距 --}}
                                <div class="eml-w-item-block" style="padding: 0px;">
                                    <div class="eml-w-phase-normal-16" style="color: red;text-align: center;font-weight:bold;font-size: 20px"> 123456 </div>
                                </div>
                            </div>

                            {{--  一级标题  --}}
                            <div class="gen-item">
                                <div class="eml-w-item-block" style="padding: 0px;">
                                    <div class="eml-w-title-level1">一、通用功能</div>
                                </div>
                            </div>

                            {{--  二级标题，以及正文--}}
                            <div class="gen-item" draggable="false">
                                <div class="eml-w-item-block" style="padding: 0px 0px 0px 1px;">
                                    {{--  二级标题--}}
                                    <div class="eml-w-title-level3">1. 管理后台 “域名管理” 优化</div>
                                    {{--  二级标题里的小字号正文  --}}
                                    <div>
                                        <div class="eml-w-phase-small-normal">
                                            <p>· 新增“管理记录”，可以查看域名注册、实名审核、续费等记录。</p>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            {{--显示图片--}}
                            <div class="gen-item" draggable="false">
                                <div class="eml-w-item-block" style="padding: 0px;">
                                    <div class="eml-w-picture-wrap">
                                        <img src="https://qy-eml-render-1258476243.cos.ap-guangzhou.myqcloud.com/images/sys/202114/pic1%402x.png"
                                             class="eml-w-picture-full-img" style="max-width: 100%;" draggable="false" onerror="">
                                    </div>
                                </div>
                            </div>

                        </div>
                    </div>
                    {{--  署名  --}}
                    <div class="eml-w-sys-footer">腾讯企业邮团队</div>
                </div>
                {{--                <img src="//exmail.qq.com/qy_mng_logic/reportKV?type=NewFeatureNotify0731&amp;itemName=NewFeatureNotify0731" style="width:1px;height:1px;display:none;" onerror="">--}}
            </div>
        </div><!--<![endif]-->
    </includetail>
</div>

<style>
    .eml-w .eml-w-phase-normal-16 {
        color: #2b2b2b;
        font-size: 16px;
        line-height: 1.75
    }

    .eml-w .eml-w-phase-bold-16 {
        font-size: 16px;
        color: #2b2b2b;
        font-weight: 500;
        line-height: 1.75
    }

    .eml-w-title-level1 {
        font-size: 20px;
        font-weight: 500;
        padding: 15px 0
    }

    .eml-w-title-level3 {
        font-size: 16px;
        font-weight: 500;
        padding-bottom: 10px
    }

    .eml-w-title-level3.center {
        text-align: center
    }

    .eml-w-phase-small-normal {
        font-size: 14px;
        color: #2b2b2b;
        line-height: 1.75
    }

    .eml-w-picture-wrap {
        padding: 10px 0;
        width: 100%;
        overflow: hidden
    }

    .eml-w-picture-full-img {
        display: block;
        width: auto;
        max-width: 100%;
        margin: 0 auto
    }

    .eml-w-sys-layout {
        background: #fff;
        box-shadow: 0 2px 8px 0 rgba(0, 0, 0, .2);
        border-radius: 4px;
        margin: 50px auto;
        max-width: 700px;
        overflow: hidden
    }

    .eml-w-sys-line-left {
        display: inline-block;
        width: 88%;
        background: #2984ef;
        height: 3px
    }

    .eml-w-sys-line-right {
        display: inline-block;
        width: 11.5%;
        height: 3px;
        background: #8bd5ff;
        margin-left: 1px
    }

    .eml-w-sys-logo {
        text-align: right
    }

    .eml-w-sys-logo img {
        display: inline-block;
        margin: 30px 50px 0 0
    }

    .eml-w-sys-content {
        position: relative;
        padding: 20px 50px 0;
        min-height: 216px;
        word-break: break-all
    }

    .eml-w-sys-footer {
        font-weight: 500;
        font-size: 12px;
        color: #bebebe;
        letter-spacing: .5px;
        padding: 0 0 30px 50px;
        margin-top: 60px
    }

    .eml-w {
        font-family: Helvetica Neue, Arial, PingFang SC, Hiragino Sans GB, STHeiti, Microsoft YaHei, sans-serif;
        -webkit-font-smoothing: antialiased;
        color: #2b2b2b;
        font-size: 14px;
        line-height: 1.75
    }

    .eml-w a {
        text-decoration: none
    }

    .eml-w a, .eml-w a:active {
        color: #186fd5
    }

    .eml-w h1, .eml-w h2, .eml-w h3, .eml-w h4, .eml-w h5, .eml-w h6, .eml-w li, .eml-w p, .eml-w ul {
        margin: 0;
        padding: 0
    }

    .eml-w-item-block {
        margin-bottom: 10px
    }

    @media (max-width: 420px) {
        .eml-w-sys-layout {
            border-radius: none !important;
            box-shadow: none !important;
            margin: 0 !important
        }

        .eml-w-sys-layout .eml-w-sys-line {
            display: none
        }

        .eml-w-sys-layout .eml-w-sys-logo img {
            margin-right: 30px !important
        }

        .eml-w-sys-layout .eml-w-sys-content {
            padding: 0 35px !important
        }

        .eml-w-sys-layout .eml-w-sys-footer {
            padding-left: 30px !important
        }
    }
</style>
