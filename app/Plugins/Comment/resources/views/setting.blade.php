<div class="card card-body">
    <div class="mb-3">
        <label class="form-label">评论最少字数</label>
        <input type="text" class="form-control" v-model="data.comment_create_min">
        <small>当前: {{get_options("comment_create_min",1)}}</small>
    </div>

    <div class="mb-3">
        <label class="form-label">评论最多字数</label>
        <input type="text" class="form-control" v-model="data.comment_create_max">
        <small>当前: {{get_options("comment_create_max",200)}}</small>
    </div>

    <div class="mb-3">
        <label class="form-label">回复最少字数</label>
        <input type="text" class="form-control" v-model="data.comment_reply_min">
        <small>当前: {{get_options("comment_reply_min",1)}}</small>
    </div>

    <div class="mb-3">
        <label class="form-label">回复最多字数</label>
        <input type="text" class="form-control" v-model="data.comment_reply_max">
        <small>当前: {{get_options("comment_reply_max",200)}}</small>
    </div>

    <div class="mb-3">
        <label class="form-label">评论间隔时间/秒</label>
        <input type="number" class="form-control" v-model="data.comment_create_time">
        <small>当前: {{get_options("comment_create_time",60)}} 秒,防灌水 建议不要低于一分钟</small>
    </div>


    <div class="mb-3">
        <label class="form-label">每次加载多少条评论?</label>
        <input type="number" class="form-control" v-model="data.comment_page_count">
        <small>当前: {{get_options("comment_page_count",15)}}</small>
    </div>

    <div class="mb-3">
        <label class="form-label">显示评论作者ip归属地</label>
        <select type="number" class="form-control" v-model="data.comment_author_ip">
            <option value="开启">开启</option>
            <option value="关闭">关闭</option>
        </select>
        <small>默认开启</small>
    </div>

    <div class="mb-3">
        <label class="form-check form-switch">
            <input class="form-check-input" type="checkbox" v-model="data.comment_ban_markdown_preview">
            <span class="form-check-label">禁用markdown预览</span>
        </label>
    </div>


</div>