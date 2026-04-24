import { type FormEvent, useState } from 'react'
import { Loader2, PackagePlus } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { createMasterItem } from '@/lib/additionalApi'
import type { MasterItemRecord } from '@/types'

type NewItemDialogProps = {
  open: boolean
  initialItemNumber?: string
  onOpenChange: (open: boolean) => void
  onCreated: (item: MasterItemRecord) => void
}

export function NewItemDialog({ open, initialItemNumber = '', onOpenChange, onCreated }: NewItemDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {open && (
        <NewItemDialogForm
          key={initialItemNumber}
          initialItemNumber={initialItemNumber}
          onCancel={() => onOpenChange(false)}
          onCreated={(item) => {
            onCreated(item)
            onOpenChange(false)
          }}
        />
      )}
    </Dialog>
  )
}

function NewItemDialogForm({
  initialItemNumber,
  onCancel,
  onCreated,
}: {
  initialItemNumber: string
  onCancel: () => void
  onCreated: (item: MasterItemRecord) => void
}) {
  const [itemNumber, setItemNumber] = useState(initialItemNumber)
  const [description, setDescription] = useState('')
  const [manufacturerKey, setManufacturerKey] = useState('')
  const [categoryKey, setCategoryKey] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!itemNumber.trim() || !description.trim() || !manufacturerKey.trim() || !categoryKey.trim()) {
      setError('itemNumber, description, manufacturerKey, categoryKey は必須です。')
      return
    }

    setIsSubmitting(true)
    setError(null)
    try {
      const created = await createMasterItem({
        itemNumber: itemNumber.trim(),
        description: description.trim(),
        manufacturerKey: manufacturerKey.trim(),
        categoryKey: categoryKey.trim(),
      })
      onCreated(created)
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'アイテム登録に失敗しました。')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <DialogContent>
      <DialogHeader>
        <DialogTitle className="flex items-center gap-2">
          <PackagePlus className="h-5 w-5" />
          新規アイテム登録
        </DialogTitle>
        <DialogDescription>
          Requirement に使う master item を登録します。登録後、このフォームへ戻って自動選択されます。
        </DialogDescription>
      </DialogHeader>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="new-item-number">itemNumber</Label>
            <Input
              id="new-item-number"
              value={itemNumber}
              onChange={(event) => setItemNumber(event.target.value)}
              placeholder="ER2"
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="new-item-manufacturer">manufacturerKey</Label>
            <Input
              id="new-item-manufacturer"
              value={manufacturerKey}
              onChange={(event) => setManufacturerKey(event.target.value)}
              placeholder="omron"
            />
          </div>
          <div className="space-y-2 sm:col-span-2">
            <Label htmlFor="new-item-description">description</Label>
            <Input
              id="new-item-description"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              placeholder="Control relay"
            />
          </div>
          <div className="space-y-2 sm:col-span-2">
            <Label htmlFor="new-item-category">categoryKey</Label>
            <Input
              id="new-item-category"
              value={categoryKey}
              onChange={(event) => setCategoryKey(event.target.value)}
              placeholder="relay"
            />
          </div>
        </div>
        {error && <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">{error}</p>}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onCancel} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button type="submit" className="gap-2" disabled={isSubmitting}>
            {isSubmitting && <Loader2 className="h-4 w-4 animate-spin" />}
            Register Item
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  )
}
