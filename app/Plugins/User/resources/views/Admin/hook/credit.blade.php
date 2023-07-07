@extends('app')
@section('title','奖励设置')
@section('content')
    <div class="row row-cards">
        <div class="col-12" id="setting-panel">
            <form action="" method="POST">
                <x-csrf/>
                <div class="card">
                    <div class="card-header">
                        <h3 class="card-title">奖励及积分设置</h3>
                    </div>
                    <div class="card-body">
                        @if(count(Itf()->get('user-admin-hook-credit')))
                            <div class="card">
                                <div class="card-header">
                                    <ul class="nav nav-tabs card-header-tabs" data-bs-toggle="tabs">

                                        @foreach(Itf()->get('user-admin-hook-credit') as $key=>$data)
                                            <li class="nav-item">
                                                <a href="#tabs-interface-{{$key}}"
                                                   class="nav-link @if($loop->first){{"active"}}@endif"
                                                   data-bs-toggle="tab">
                                                    <!-- Download SVG icon from http://tabler-icons.io/i/user -->
                                                    {!! $data['name'] !!}
                                                </a>
                                            </li>
                                        @endforeach
                                    </ul>
                                </div>
                                <div class="card-body">
                                    <div class="tab-content">
                                        @foreach(Itf()->get('user-admin-hook-credit') as $key=>$data)
                                            <div class="tab-pane @if($loop->first){{"active show"}}@endif"
                                                 id="tabs-interface-{{$key}}">
                                                @include($data['view'])
                                            </div>
                                        @endforeach
                                    </div>
                                    @else
                                        <div class="empty">
                                            <div class="empty-header">403</div>
                                            <p class="empty-title">暂无接口</p>
                                            <p class="empty-subtitle text-muted">
                                                暂无可用的发信服务接口，请尝试自己扩展或安装扩展插件
                                            </p>
                                        </div>
                                    @endif
                                </div>
                                @if(count(Itf()->get('user-admin-hook-credit')))
                                    <div class="card-footer">
                                        <div class="row">
                                            <div class="col"></div>
                                            <div class="col-auto">
                                                <button class="btn btn-primary">保存</button>
                                            </div>
                                        </div>
                                    </div>
                                @endif
                            </div>
                    </div>
                </div>
            </form>
        </div>
    </div>
@endsection
@section('scripts')
    <script>
        $(function (){
            $('input[type="checkbox"]').each(function (){
                // 在此元素下面创建name相同的input hidden
                $(this).after('<input type="hidden" name="' + this.name + '" value="' + this.value + '">')
                // 修改当前元素的属性，把name=* 改为 bs-name=*
                $(this).attr('bs-name', this.name)
                // 删除当前元素的name属性
                $(this).removeAttr('name')
                // 判断checkbox未被勾选
                if(!$(this).is(':checked')) {
                    // 获取当前元素bs-name
                    let name = $(this).attr('bs-name')
                    // 获取当前元素的同级input hidden
                    let hidden = $(this).siblings('input[type="hidden"][name="' + name + '"]')
                    // 为input hidden赋值
                    hidden.val("false")
                } else {
                    // 获取当前元素bs-name
                    let name = $(this).attr('bs-name')
                    // 获取当前元素的同级input hidden
                    let hidden = $(this).siblings('input[type="hidden"][name="' + name + '"]')
                    // 为input hidden赋值
                    hidden.val("true")
                }
            })

            // 监听所有checkbox的变化
            $('input[type="checkbox"]').change(function (){
                // 判断checkbox未被勾选
                if(!$(this).is(':checked')) {
                    // 获取当前元素bs-name
                    let name = $(this).attr('bs-name')
                    // 获取当前元素的同级input hidden
                    let hidden = $(this).siblings('input[type="hidden"][name="' + name + '"]')
                    // 为input hidden赋值
                    hidden.val("false")
                } else {
                    // 获取当前元素bs-name
                    let name = $(this).attr('bs-name')
                    // 获取当前元素的同级input hidden
                    let hidden = $(this).siblings('input[type="hidden"][name="' + name + '"]')
                    // 为input hidden赋值
                    hidden.val("true")
                }
            })
        })
    </script>
@endsection