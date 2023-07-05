<div class="card card-body">

    <div class="row row-cards">
        <div class="col-lg-4">
            <div class="form-label">Gavatar镜像源</div>
            <select v-model="data.theme_common_gavatar" class="form-select" >
                <option value="https://www.gravatar.com/avatar/">官方</option>
                <option value="https://cn.gravatar.com/avatar/">官方cn源</option>
                <option value="https://gravatar.loli.net/avatar/">loli.net</option>
                <option value="https://sdn.geekzu.org/avatar/">极客族</option>
                <option value="https://cdn.v2ex.com/gravatar/">V2EX</option>
                <option value="https://dn-qiniu-avatar.qbox.me/avatar/">七牛云</option>
            </select>
        </div>

        <div class="col-lg-4">
            <div class="form-label">网站icon</div>
            <input type="text" class="form-control" v-model="data.theme_common_icon">
            <small>填写链接,<a href="/admin/files/upload" target="_blank">点我</a> 上传文件</small>
        </div>
        <div class="col-lg-4">
            <div class="form-label">友情链接</div>
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.theme_common_friend_links">
                <span class="form-check-label">开启友链功能</span>
            </label>
        </div>

        <div class="col-lg-4">
            <div class="form-label">黏性导航栏</div>
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.theme_common_navbar_sticky">
                <span class="form-check-label">开启黏性导航栏</span>
            </label>
        </div>

        <div class="col-lg-4">
            <div class="form-label">友链显示位置</div>
            <select v-model="data.theme_common_friend_links_position" class="form-select">
                <option value="home">首页</option>
                <option value="common">全局</option>
            </select>
            <small>默认只在首页显示</small>
        </div>
        <div class="col-lg-4">
            <div class="form-label">申请友链地址</div>
            <input type="text" v-model="data.theme_common_friend_links_apply" class="form-control">
        </div>
        <div class="col-lg-4">
            <div class="form-label">查看全部友链地址</div>
            <input type="text" v-model="data.theme_common_friend_links_all" class="form-control">
        </div>
        <div class="col-lg-4">
            <div class="form-label">head标签内自定义代码</div>
            <input type="text" v-model="data.theme_common_diy_code_head" class="form-control">
            <small>请输入小部件调用代码, <a href="/admin/hook/components" target="_blank">点我进入小部件页面</a> </small>
        </div>
        <div class="col-lg-4">
            <div class="form-label">body标签结尾自定义代码</div>
            <input type="text" v-model="data.theme_common_diy_code_body" class="form-control">
            <small>请输入小部件调用代码, <a href="/admin/hook/components" target="_blank">点我进入小部件页面</a> </small>
        </div>

        <div class="col-lg-4 align-self-center">
            <div class="form-label">首页移动端显示标签图标</div>
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.theme_home_tag_icon">
                <span class="form-check-label">开启</span>
            </label>
        </div>

        <div class="col-lg-4 align-self-center">
            <div class="form-label">右侧边栏小工具黏性</div>
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.theme_right_tool_sticky">
                <span class="form-check-label">关闭</span>
            </label>
        </div>

        <div class="col-lg-4 align-self-center">
            <div class="form-label">首页文章标题自动截断</div>
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.theme_home_title_truncate">
                <span class="form-check-label">开启</span>
            </label>
        </div>
    </div>

</div>