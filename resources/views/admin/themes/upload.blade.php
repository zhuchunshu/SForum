@extends("app")

@section('title','上传主题')
@section('content')
    <div class="col-md-12">
        <div class="row row-cards">
            <div class="col-md-6">
                <form method="post" action="/admin/themes/upload" enctype="multipart/form-data">
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
                <form method="post" action="/admin/themes/download">
                    <x-csrf/>
                    <div class="card">
                        <div class="card-body">
                            <h3 class="card-title">远程上传(安全性问题,暂不支持)</h3>
                            <div class="mb-3">
                                <label for="" class="form-label">请填写主题.zip或.tar.gz下载链接</label>
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