<div class="card card-body">
    <div class="row">
        <div class="col-4">
            <label class="form-label">默认头像源
            </label>
            <select class="form-select" v-model="data.core_user_def_avatar">
                <option value="gavatar">Gavatar</option>
                <option value="ui-avatars">ui-avatars</option>
            </select>
            <small>默认使用Gravatar的头像</small>
        </div>

        <div class="col-1">
            <label class="form-label">ui-avatars
                <div>
                    <span class="avatar" style="background-image:url(https://ui-avatars.com/api/?background=random&name=Zhu+Chunshu)"></span>
                </div>
            </label>
        </div>

        <div class="col-4">
            <label class="form-label">头像缓存
            </label>
            <select class="form-select" v-model="data.core_user_avatar_cache">
                <option value="1">开启</option>
                <option value="2">关闭</option>
            </select>
            <small>默认开启</small>
        </div>

        <div class="col-3">
            <label class="form-label">强制验证邮箱
            </label>
            <select class="form-select" v-model="data.core_user_email_ver">
                <option value="1">开启</option>
                <option value="2">关闭</option>
            </select>
            <small>默认开启</small>
        </div>


        <div class="col-3">
            <label class="form-label">头像缓存时间
            </label>
            <input type="number" class="form-control" v-model="data.core_user_def_avatar_cache">
            <small>头像缓存时间/秒,减少查询,大幅度优化速度 默认10分钟</small>
        </div>

        <div class="col-3">
            <label class="form-label">图片上传大小限制 / KB
            </label>
            <input type="number" class="form-control" v-model="data.core_user_up_img_size">
            <small>当前:{{get_options('core_user_up_img_size',2048)}} KB</small>
        </div>

        <div class="col-3">
            <label class="form-label">文件上传大小限制 / KB
            </label>
            <input type="number" class="form-control" v-model="data.core_user_up_file_size">
            <small>当前:{{get_options('core_user_up_file_size',4096)}} KB</small>
        </div>

        <div class="col-3">
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.user_email_noticed_on">
                <span class="form-check-label">开启所有用户邮件通知功能(需用户自行手动关闭)</span>
            </label>
        </div>

    </div>

</div>