@extends('app')
@section('title','EditFile')

@section('content')
    <style type="text/css" media="screen">
        #editor {
            width: 100%;
            min-height: 600px;
            font-size:20px;
            margin-bottom:10px
        }
    </style>
    <div id="EditFile">
        <div class="row">
            <div class="col-md-12">
                <div class="row">
                    <div class="col"><h3>代码编辑器</h3></div>
                    <div class="col-auto">
                        {{$title}}
                    </div>
                </div>
            </div>
            <div class="col-md-12">
                <div id="editor">
                    {{$content}}
                </div>
            </div>
            <div class="col-md-12">
                <button @@click="submit()" class="btn btn-primary">保存</button>
            </div>
        </div>
    </div>
@endsection


@section('scripts')
    <script type="text/javascript" src="/js/ace/ace.js"></script>
    <script type="text/javascript">
        const lang = "{{$lang}}";
        const action ="{{$action}}";
    </script>
    <script src="{{mix('js/admin/EditFile.js')}}"></script>
@endsection
