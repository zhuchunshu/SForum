<div class="card card-body">
    <div class="mb-3">
        <div class="row">
            <div class="col-6">
                <label class="form-label">是否开启渲染twemoji</label>
                <select class="form-control" v-model="data.contentParse_twemoji">
                    <option value="开启">maxcdn</option>
                    <option value="关闭">关闭</option>
                </select>
                <small>当前: {{get_options("contentParse_twemoji",'开启')}}</small>
            </div>
            <div class="col-6">
                <label class="form-label">twemoji静态资源库</label>
                <select class="form-control" v-model="data.contentParse_twemoji_cdn">
                    <option value="https://twemoji.maxcdn.com/v/latest">maxcdn</option>
                    <option value="https://lib.baomitu.com/twemoji/1.4.2">baomitu</option>
                    <option value="https://lf26-cdn-tos.bytecdntp.com/cdn/expire-1-M/twemoji/13.1.0">字节跳动(版本低)</option>
                </select>
                <small>默认: baomitu</small>
            </div>
        </div>
    </div>

    <div class="mb-3">
        <div class="row">
            <div class="col-6">
                <label class="form-label">twemoji 图片宽度</label>
                <input type="number" class="form-control" min="1" v-model="data.contentParse_twemoji_contentParse_width">
                <small>当前: {{get_options("contentParse_twemoji_contentParse_width",25)}}</small>
            </div>
            <div class="col-6">
                <label class="form-label">twemoji 图片高度</label>
                <input type="number" class="form-control" min="1" v-model="data.contentParse_twemoji_contentParse_height">
                <small>当前: {{get_options("contentParse_twemoji_contentParse_height",25)}}</small>
            </div>
        </div>
    </div>


</div>