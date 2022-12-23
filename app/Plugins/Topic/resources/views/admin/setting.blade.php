<div class="card card-body">
    <div class="mb-3">
        <label class="form-label">标题最少要求字数</label>
        <input type="text" class="form-control" v-model="data.topic_create_title_min">
        <small>当前: {{get_options("topic_create_title_min",1)}}</small>
    </div>

    <div class="mb-3">
        <label class="form-label">标题最多要求字数</label>
        <input type="text" class="form-control" v-model="data.topic_create_title_max">
        <small>当前: {{get_options("topic_create_title_max",200)}}</small>
    </div>

    <div class="mb-3">
        <label class="form-label">正文内容最少要求字数</label>
        <input type="text" class="form-control" v-model="data.topic_create_content_min">
        <small>当前: {{get_options("topic_create_content_min",10)}}</small>
    </div>

    <div class="mb-3">
        <label class="form-label">发帖间隔时间/秒</label>
        <input type="number" class="form-control" v-model="data.topic_create_time">
        <small>当前: {{get_options("topic_create_time",120)}} 秒,防刷帖 建议不要低于一分钟</small>
    </div>

    <div class="mb-3">
        <label class="form-label">每页显示帖子数量</label>
        <input type="number" class="form-control" v-model="data.topic_home_num">
        <small>当前: {{get_options("topic_home_num",15)}}</small>
    </div>

    <div class="mb-3">
        <label class="form-label">显示帖子作者ip归属地</label>
        <select type="number" class="form-control" v-model="data.topic_author_ip">
            <option value="开启">开启</option>
            <option value="关闭">关闭</option>
        </select>
        <small>默认开启</small>
    </div>

    <div class="mb-3">
        <label class="form-label">显示帖子修订者ip归属地</label>
        <select class="form-control" v-model="data.topic_updated_author_ip">
            <option value="开启">开启</option>
            <option value="关闭">关闭</option>
        </select>
        <small>默认开启</small>
    </div>

    <div class="mb-3">
        <label class="form-label">摘要长度</label>
        <input type="number" class="form-control" mix="1" v-model="data.topic_brief_len">
        <small>默认250</small>
    </div>

    <div class="mb-3">
        <label class="form-check form-switch">
            <input class="form-check-input" type="checkbox" v-model="data.topic_emoji_close">
            <span class="form-check-label">关闭插入表情功能</span>
        </label>
    </div>

    <div class="mb-3">
        <label class="form-check form-switch">
            <input class="form-check-input" type="checkbox" v-model="data.topic_create_tag_ex">
            <span class="form-check-label">创建标签需要审核</span>
        </label>
    </div>


</div>