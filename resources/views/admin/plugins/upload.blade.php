@extends("app")

@section('title','上传插件')
@section('content')
    <div class="col-md-12">
        <div class="row row-cards">
            <div class="col-12">
                <div class="alert alert-info alert-dismissible" role="alert">
                    <div class="d-flex">
                        <div>
                            <!-- Download SVG icon from http://tabler-icons.io/i/info-circle -->
                            <svg xmlns="http://www.w3.org/2000/svg" class="icon alert-icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><circle cx="12" cy="12" r="9" /><line x1="12" y1="8" x2="12.01" y2="8" /><polyline points="11 12 12 12 12 16 13 16" /></svg>
                        </div>
                        <div>
                            <div class="text-muted">安装插件不只是上传即可，如果第一次使用请<a href="https://sforum.cn/use/instructions/plugin.html">点击这里</a>查看插件使用教程 </div>
                        </div>
                    </div>
                    <a class="btn-close" data-bs-dismiss="alert" aria-label="close"></a>
                </div>
            </div>
            <div class="col-md-6">
                <form method="post" action="/admin/plugins/upload" enctype="multipart/form-data">
                    <x-csrf/>
                    <div class="card">
                        <div class="card-body">
                            <h3 class="card-title">本地上传</h3>
                            <div class="mb-3">
                                <label for="" class="form-label">请选择上传.zip或.tar.gz文件</label>
                                <input accept=".zip , .tar.gz" type="file" name="file" class="form-control" required>
                            </div>
                            <div class="mb-3">
                                <button class="btn btn-dark">上传</button>
                            </div>
                        </div>
                    </div>
                </form>
            </div>

            <div class="col-md-6">
                <form method="post" action="/admin/plugins/download">
                    <x-csrf/>
                    <div class="card">
                        <div class="card-body">
                            <h3 class="card-title">远程上传(安全性问题,暂不支持)</h3>
                            <div class="mb-3">
                                <label for="" class="form-label">请填写插件.zip或.tar.gz下载链接</label>
                                <input disabled type="url" name="file" class="form-control" required>
                            </div>
                            <div class="mb-3">
                                <button disabled class="btn btn-dark">
                                    提交
                                </button>
                            </div>
                        </div>
                    </div>
                </form>
            </div>
        </div>
    </div>
@endsection