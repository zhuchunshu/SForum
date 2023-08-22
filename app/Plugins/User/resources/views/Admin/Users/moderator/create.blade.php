@extends('app')
@section('title','新增版主')
@section('content')

    <div class="row row-cards">
        <div class="col-12">
            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">新增版主</h3>
                    <div class="card-actions">
                        <a href="../moderator">版主列表</a>
                    </div>
                </div>

                <div class="card-body">
                    <form action="" method="post">
                        <x-csrf/>
                        <div class="mb-3">
                            <label class="form-label required">选择用户</label>
                            <select type="text" class="form-select" placeholder="Select a user" name="user_id" id="select-people" required>
                                <option value="0">请选择</option>
                                @foreach(App\Plugins\User\src\Models\User::all(['id','class_id','avatar','email','username']) as $user)
                                    <option value="{{$user->id}}" data-custom-properties="&lt;span class=&quot;avatar avatar-xs&quot; style=&quot;background-image: url({{avatar($user)}})&quot;&gt;&lt;/span&gt;">
                                        {{$user->username}}</option>
                                @endforeach
                            </select>
                        </div>

                        <div class="mb-3">
                            <label class="form-label required">选择论坛(板块)</label>
                            <select type="text" class="form-select" placeholder="Select a user" name="tag_id" id="select-tag" required>
                                @foreach(\App\Plugins\Topic\src\Models\TopicTag::all() as $tag)
                                    <option value="0">请选择</option>
                                    <option value="{{$tag->id}}" data-custom-properties="{{'<span class="badge bg-primary-lt">'.$tag->icon.'</span>'}}">
                                        {{$tag->name}}</option>
                                @endforeach
                            </select>
                        </div>
                        <div class="mb-1">
                            <button type="submit" class="btn btn-primary">提交</button>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    </div>

@endsection

@section('scripts')
    <script src="{{file_hash('tabler/libs/tom-select/dist/js/tom-select.base.min.js')}}"></script>
    <script>
        // @formatter:off
        document.addEventListener("DOMContentLoaded", function () {
            var el;
            window.TomSelect && (new TomSelect(el = document.getElementById('select-people'), {
                copyClassesToDropdown: false,
                dropdownClass: 'dropdown-menu ts-dropdown',
                optionClass:'dropdown-item',
                controlInput: '<input>',
                render:{
                    item: function(data,escape) {
                        if( data.customProperties ){
                            return '<div><span class="dropdown-item-indicator">' + data.customProperties + '</span>' + escape(data.text) + '</div>';
                        }
                        return '<div>' + escape(data.text) + '</div>';
                    },
                    option: function(data,escape){
                        if( data.customProperties ){
                            return '<div><span class="dropdown-item-indicator">' + data.customProperties + '</span>' + escape(data.text) + '</div>';
                        }
                        return '<div>' + escape(data.text) + '</div>';
                    },
                },
            }));
        });
        // @formatter:on

        // @formatter:off
        document.addEventListener("DOMContentLoaded", function () {
            var el;
            window.TomSelect && (new TomSelect(el = document.getElementById('select-tag'), {
                copyClassesToDropdown: false,
                dropdownClass: 'dropdown-menu ts-dropdown',
                optionClass:'dropdown-item',
                controlInput: '<input>',
                render:{
                    item: function(data,escape) {
                        if( data.customProperties ){
                            return '<div><span class="dropdown-item-indicator">' + data.customProperties + '</span>' + escape(data.text) + '</div>';
                        }
                        return '<div>' + escape(data.text) + '</div>';
                    },
                    option: function(data,escape){
                        if( data.customProperties ){
                            return '<div><span class="dropdown-item-indicator">' + data.customProperties + '</span>' + escape(data.text) + '</div>';
                        }
                        return '<div>' + escape(data.text) + '</div>';
                    },
                },
            }));
        });
        // @formatter:on
    </script>
@endsection