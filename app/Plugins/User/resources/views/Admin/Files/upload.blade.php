@extends("app")

@section('title',"文件上传")


@section('content')
    <div class="col-md-12">
        <div class="border-0 card">
            <div class="card-body">
                <h3 class="card-title">文件上传</h3>
                <form action="?Redirect=/admin/files/upload" method="post" enctype="multipart/form-data">
                    <x-csrf/>
                    <div class="mb-3">
                        <input type="file" class="form-control" id="file" name="file">
                    </div>
                    <button type="submit" class="btn btn-primary">上传</button>
                </form>
            </div>
        </div>
    </div>

    @if(request()->has('url'))
        <div class="col-md-12">
            <div class="border-0 card">
                <div class="card-body">
                    <h3 class="card-title">文件上传成功!</h3>
                    <div class="mb-3">
                        <label for="" class="form-label">文件链接</label>
                        @if(\Hyperf\Stringable\Str::is('http*//*',request()->input("url")))
                            <input type="text" class="form-control" value="{{(request()->input("url"))}}">
                        @else
                            <input type="text" class="form-control" value="{{url(request()->input("url"))}}">
                        @endif
                    </div>
                </div>
            </div>
        </div>
    @endif
@endsection
