<?php

declare (strict_types=1);
namespace App\Plugins\User\src\Models;

use App\Model\Model;

/**
 * @property string $user_id 
 * @property string $name 
 * @property string $value 
 */
class UsersSetting extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'users_settings';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['user_id','name','value'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = [];
	
	public $timestamps = false;
}