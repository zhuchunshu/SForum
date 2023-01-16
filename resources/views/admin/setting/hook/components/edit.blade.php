@extends('app')

@section('title','编辑代码')
@section('content')

    <style type="text/css" media="screen">
        #editor {
            width: 100%;
            min-height: 600px;
            font-size:20px;
            margin-bottom:10px;
        }
    </style>

    <div class="row row-cards">
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">修改小部件代码</h3>
                <div class="card-actions">
                    <a href="/admin/hook/components"> 返回部件列表</a>
                </div>
            </div>
            <div class="card-body" id="vue-admin-hook-component-edit">
                <div id="editor">加载失败...</div>
                <div class="mt-3 row">
                    <div class="col"></div>
                    <div class="col-auto">
                        <button class="btn btn-primary" @@click="submit">保存</button>
                    </div>
                </div>
            </div>
        </div>
    </div>

@endsection
@section('scripts')
    <script type="text/javascript" src="/js/ace/ace.js"></script>
    <script type="text/javascript" src="/js/ace/ext-language_tools.js"></script>
    <script type="text/javascript">
        const get_action = "/admin/hook/components/get_file_content?path={{$path}}"
        const put_action = "/admin/hook/components/put_file_content?path={{$path}}"
        const lang = "php_laravel_blade";
    </script>
    <script src="{{mix('js/admin/component.js')}}"></script>
@endsection
