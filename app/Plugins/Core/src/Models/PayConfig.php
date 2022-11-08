<?php

declare (strict_types=1);
namespace App\Plugins\Core\src\Models;

use App\Model\Model;

/**
 * @property string $name 
 * @property string $value 
 */
class PayConfig extends Model
{
    public $timestamps = false;
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'pay_config';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['name','value'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = [];
}